// WebP変換用Lambda関数（シンプル版）
import { SQSClient, SendMessageCommand } from '@aws-sdk/client-sqs';
import axios from 'axios';
import sharp from 'sharp';
import fs from 'fs/promises';
import path from 'path';

// Lambda関数のハンドラー
export const handler = async (event) => {
  console.log('WebP変換処理を開始します');
  
  // 環境変数からAPIキーとURLを取得
  const CLOUDFLARE_WORKER_URL = process.env.CLOUDFLARE_WORKER_URL;
  const API_KEY = process.env.API_KEY;
  const RESULT_QUEUE_URL = process.env.RESULT_QUEUE_URL;
  
  // 設定確認
  if (!CLOUDFLARE_WORKER_URL || !API_KEY) {
    console.error('環境変数の設定が不足しています: CLOUDFLARE_WORKER_URL, API_KEY');
    return {
      statusCode: 500,
      body: JSON.stringify({
        status: 'error',
        message: '環境変数の設定が不足しています'
      })
    };
  }
  
  // SQSクライアントの初期化
  const sqsClient = RESULT_QUEUE_URL ? new SQSClient({
    region: process.env.AWS_REGION || 'ap-northeast-1'
  }) : null;
  
  // SQSからのイベントを処理
  if (event.Records && event.Records.length > 0) {
    const results = [];
    
    for (const record of event.Records) {
      try {
        const messageBody = JSON.parse(record.body);
        console.log('処理するメッセージ:', messageBody);
        
        // 画像変換処理
        if (messageBody.type === 'webp_conversion' && messageBody.imageData) {
          const result = await processImage(
            messageBody.imageData,
            CLOUDFLARE_WORKER_URL,
            API_KEY,
            sqsClient,
            RESULT_QUEUE_URL
          );
          results.push(result);
        }
      } catch (error) {
        console.error('メッセージ処理中にエラーが発生しました:', error);
        results.push({
          status: 'error',
          message: error.message
        });
      }
    }
    
    return {
      statusCode: 200,
      body: JSON.stringify({
        results: results
      })
    };
  } else {
    // 直接呼び出しの場合
    try {
      if (event.imageData) {
        const result = await processImage(
          event.imageData,
          CLOUDFLARE_WORKER_URL,
          API_KEY,
          sqsClient,
          RESULT_QUEUE_URL
        );
        return {
          statusCode: 200,
          body: JSON.stringify(result)
        };
      } else {
        return {
          statusCode: 400,
          body: JSON.stringify({
            status: 'error',
            message: '画像データが必要です'
          })
        };
      }
    } catch (error) {
      console.error('処理中にエラーが発生しました:', error);
      return {
        statusCode: 500,
        body: JSON.stringify({
          status: 'error',
          message: error.message
        })
      };
    }
  }
};

// 単一画像の処理
async function processImage(imageData, cloudflareWorkerUrl, apiKey, sqsClient, resultQueueUrl) {
  console.log(`画像を処理します: ${imageData.fileName}`);
  
  try {
    // 一時ファイル名の設定
    const tempDir = '/tmp';
    const tempInputFile = `${tempDir}/${imageData.fileName}`;
    const fileExt = path.extname(imageData.fileName).toLowerCase();
    const baseName = path.basename(imageData.fileName, fileExt);
    const webpOutputFileName = `${baseName}.webp`;
    const tempWebpFile = `${tempDir}/${webpOutputFileName}`;
    
    // Cloudflare R2から画像を取得
    console.log('CloudflareワーカーからR2の画像を取得します');
    const r2Response = await axios({
      method: 'GET',
      url: `${cloudflareWorkerUrl}/file/${imageData.originalPath}`,
      responseType: 'arraybuffer',
      headers: {
        'X-API-Key': apiKey
      },
      timeout: 30000 // 30秒タイムアウト
    });
    
    // 画像を一時ファイルに保存
    await fs.writeFile(tempInputFile, Buffer.from(r2Response.data));
    console.log('画像を一時ファイルに保存しました');
    
    // sharpを使用して画像メタデータを取得
    const metadata = await sharp(tempInputFile).metadata();
    console.log(`元の画像サイズ: ${metadata.width}x${metadata.height}`);
    
    // sharpを使用してWebP変換とリサイズを行う
    let sharpPipeline = sharp(tempInputFile);
    
    // 大きな画像の場合はリサイズ
    if (metadata.width > 1600 || metadata.height > 1600) {
      // アスペクト比を維持したリサイズ
      sharpPipeline = sharpPipeline.resize({
        width: Math.min(metadata.width, 1600),
        height: Math.min(metadata.height, 1600),
        fit: 'inside',
        withoutEnlargement: true
      });
    }
    
    // WebP形式に変換
    const webpBuffer = await sharpPipeline.webp({
      quality: 80,
      lossless: false,
      alphaQuality: 90,
      effort: 4
    }).toBuffer();
    
    // 変換後のWebP画像を一時ファイルに保存
    await fs.writeFile(tempWebpFile, webpBuffer);
    console.log('WebP変換が完了しました');
    
    // 元のファイルとWebPのサイズを比較
    const originalSize = Buffer.byteLength(r2Response.data);
    const webpSize = webpBuffer.length;
    const compressionRatio = ((originalSize - webpSize) / originalSize * 100).toFixed(2);
    console.log(`圧縮率: ${compressionRatio}% (元: ${originalSize}バイト, WebP: ${webpSize}バイト)`);
    
    // Cloudflare WorkerにWebP画像をアップロード
    console.log('変換されたWebP画像をR2にアップロードします');
    
    // FormDataを作成
    const formData = new FormData();
    formData.append('file', new Blob([webpBuffer]), webpOutputFileName);
    formData.append('type', 'webp');
    formData.append('fileName', webpOutputFileName);
    
    const uploadResponse = await axios.post(
      `${cloudflareWorkerUrl}/upload`,
      formData,
      {
        headers: {
          'X-API-Key': apiKey,
          'Content-Type': 'multipart/form-data'
        },
        timeout: 30000 // 30秒タイムアウト
      }
    );
    
    // 一時ファイルを削除
    try {
      await fs.unlink(tempInputFile);
      await fs.unlink(tempWebpFile);
      console.log('一時ファイルを削除しました');
    } catch (err) {
      console.error('一時ファイルの削除に失敗しました:', err);
    }
    
    // 結果オブジェクト
    const result = {
      status: 'success',
      imageId: imageData.id,
      webpPath: uploadResponse.data.path,
      stats: {
        originalSize,
        webpSize,
        compressionRatio: parseFloat(compressionRatio),
        dimensions: {
          width: metadata.width,
          height: metadata.height
        }
      }
    };
    
    // 結果をSQSに送信（キューURLが設定されている場合）
    if (sqsClient && resultQueueUrl) {
      await sendResultToSQS(sqsClient, resultQueueUrl, imageData.id, result);
    }
    
    return result;
  } catch (error) {
    console.error('画像処理中にエラーが発生しました:', error);
    
    // エラー結果
    const errorResult = {
      status: 'error',
      imageId: imageData.id,
      message: error.message
    };
    
    // エラー結果をSQSに送信（キューURLが設定されている場合）
    if (sqsClient && resultQueueUrl) {
      await sendResultToSQS(sqsClient, resultQueueUrl, imageData.id, errorResult);
    }
    
    throw error;
  }
}

// 結果をSQSに送信
async function sendResultToSQS(sqsClient, queueUrl, imageId, result) {
  try {
    const params = {
      QueueUrl: queueUrl,
      MessageBody: JSON.stringify({
        type: 'webp_conversion_result',
        imageData: {
          id: imageId,
          result: result
        }
      })
    };
    
    await sqsClient.send(new SendMessageCommand(params));
    console.log(`結果をSQSに送信しました: ${queueUrl}`);
  } catch (error) {
    console.error('SQSへの結果送信に失敗しました:', error);
  }
}