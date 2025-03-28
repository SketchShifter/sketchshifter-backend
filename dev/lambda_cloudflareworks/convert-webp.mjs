// WebP変換用Lambda関数 (完全無料化版)
// - AWS SDK v3 使用
// - ESモジュール形式
// - Systems Manager Parameter Store (標準パラメータ) 使用
// - mysql2で直接RDS接続
// - 保存先: lambda/webp-converter/index.mjs

import { SQSClient, SendMessageCommand } from '@aws-sdk/client-sqs';
import { SSMClient, GetParameterCommand } from '@aws-sdk/client-ssm';
import axios from 'axios';
import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { promisify } from 'util';
import mysql from 'mysql2/promise';

const writeFile = promisify(fs.writeFile);
const readFile = promisify(fs.readFile);
const unlink = promisify(fs.unlink);
const stat = promisify(fs.stat);

// AWS クライアントの初期化
const sqs = new SQSClient({ region: process.env.AWS_REGION || 'ap-northeast-1' });
const ssm = new SSMClient({ region: process.env.AWS_REGION || 'ap-northeast-1' });

// キャッシュされたパラメータ
let cachedParams = null;
let paramsLastFetchTime = 0;
const PARAM_CACHE_TTL = 15 * 60 * 1000; // 15分

// 圧縮設定
const COMPRESSION_SETTINGS = {
  // WebP設定
  webp: {
    quality: 80,        // 品質 (0-100)
    lossless: false,    // ロスレス圧縮を使用しない
    alphaQuality: 90,   // アルファチャンネルの品質 (0-100)
    effort: 4,          // 圧縮の努力レベル (0-6)
    resize: {
      enabled: true,    // リサイズを有効にする
      maxWidth: 1600,   // 最大幅
      maxHeight: 1600   // 最大高さ
    }
  }
};

// データベース接続を取得
async function getDbConnection(config) {
  return await mysql.createConnection({
    host: config.RDS_HOST || process.env.RDS_HOST,
    port: parseInt(config.RDS_PORT || process.env.RDS_PORT || '3306', 10),
    user: config.RDS_USER || process.env.RDS_USER,
    password: config.RDS_PASSWORD || process.env.RDS_PASSWORD,
    database: config.RDS_DATABASE || process.env.RDS_DATABASE,
    connectTimeout: 10000,
    waitForConnections: true
  });
}

// Systems Managerパラメータストアから設定を取得
async function getParameters() {
  // キャッシュが有効なら使用
  const now = Date.now();
  if (cachedParams && now - paramsLastFetchTime < PARAM_CACHE_TTL) {
    return cachedParams;
  }

  try {
    // パラメータパスをカスタマイズ
    const paramPath = process.env.PARAM_PATH || '/sketchshifter/config';
    
    const command = new GetParameterCommand({
      Name: paramPath,
      WithDecryption: true
    });
    
    const response = await ssm.send(command);
    
    // パラメータ値をJSONとして解析
    const config = JSON.parse(response.Parameter.Value);
    
    // 環境変数とマージした設定
    cachedParams = {
      CLOUDFLARE_WORKER_URL: config.cloudflareWorkerUrl,
      API_KEY: config.apiKey,
      RDS_HOST: config.rdsHost || process.env.RDS_HOST,
      RDS_PORT: config.rdsPort || process.env.RDS_PORT || '3306',
      RDS_USER: config.rdsUser || process.env.RDS_USER,
      RDS_PASSWORD: config.rdsPassword || process.env.RDS_PASSWORD,
      RDS_DATABASE: config.rdsDatabase || process.env.RDS_DATABASE
    };
    
    paramsLastFetchTime = now;
    return cachedParams;
  } catch (error) {
    console.error('パラメータの取得中にエラーが発生しました:', error);
    
    // エラーが発生した場合は環境変数のみを使用
    return {
      CLOUDFLARE_WORKER_URL: process.env.CLOUDFLARE_WORKER_URL,
      API_KEY: process.env.API_KEY,
      RDS_HOST: process.env.RDS_HOST,
      RDS_PORT: process.env.RDS_PORT || '3306',
      RDS_USER: process.env.RDS_USER,
      RDS_PASSWORD: process.env.RDS_PASSWORD,
      RDS_DATABASE: process.env.RDS_DATABASE
    };
  }
}

export const handler = async (event) => {
  console.log('WebP変換処理を開始します');
  
  // 設定を取得
  const config = await getParameters();
  
  // SQSからのイベントを処理
  if (event.Records && event.Records.length > 0) {
    const results = [];
    
    for (const record of event.Records) {
      try {
        const messageBody = JSON.parse(record.body);
        console.log('処理するメッセージ:', messageBody);
        
        if (messageBody.type === 'webp_conversion') {
          // 単一の画像変換処理
          const result = await processImage(messageBody.imageData, config);
          results.push(result);
        } else if (messageBody.type === 'batch_conversion') {
          // バッチ処理のトリガー
          const batchResult = await processBatch(messageBody.batchSize || 10, config);
          results.push(batchResult);
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
    // スケジュールイベントからの実行（バッチ処理）
    console.log('スケジュールイベントからバッチ処理を開始します');
    
    try {
      const result = await processBatch(20, config);
      return {
        statusCode: 200,
        body: JSON.stringify(result)
      };
    } catch (error) {
      console.error('バッチ処理中にエラーが発生しました:', error);
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
async function processImage(imageData, config) {
  console.log(`画像を処理します: ${imageData.fileName}`);
  
  try {
    // 一時ファイル名の設定
    const tempInputFile = `/tmp/${imageData.fileName}`;
    const fileExt = path.extname(imageData.fileName).toLowerCase();
    const baseName = path.basename(imageData.fileName, fileExt);
    const webpOutputFileName = `${baseName}.webp`;
    const tempWebpFile = `/tmp/${webpOutputFileName}`;
    
    // Cloudflare R2から画像を取得
    console.log('CloudflareワーカーからR2の画像を取得します');
    const r2Response = await axios({
      method: 'GET',
      url: `${config.CLOUDFLARE_WORKER_URL}/file/original/${imageData.fileName}`,
      responseType: 'arraybuffer',
      headers: {
        'X-API-Key': config.API_KEY
      }
    });
    
    // 画像を一時ファイルに保存
    await writeFile(tempInputFile, r2Response.data);
    console.log('画像を一時ファイルに保存しました');
    
    // sharpを使用して画像メタデータを取得
    const metadata = await sharp(tempInputFile).metadata();
    console.log(`元の画像サイズ: ${metadata.width}x${metadata.height}`);
    
    // sharpを使用してWebP変換とリサイズを行う
    let sharpPipeline = sharp(tempInputFile);
    
    // リサイズが必要か判断
    const settings = COMPRESSION_SETTINGS.webp;
    if (settings.resize.enabled &&
        (metadata.width > settings.resize.maxWidth || metadata.height > settings.resize.maxHeight)) {
      // アスペクト比を維持したリサイズ
      sharpPipeline = sharpPipeline.resize({
        width: Math.min(metadata.width, settings.resize.maxWidth),
        height: Math.min(metadata.height, settings.resize.maxHeight),
        fit: 'inside',
        withoutEnlargement: true
      });
    }
    
    // WebP形式に変換
    const webpBuffer = await sharpPipeline.webp({
      quality: settings.quality,
      lossless: settings.lossless,
      alphaQuality: settings.alphaQuality,
      effort: settings.effort
    }).toBuffer();
    
    // 変換後のWebP画像を一時ファイルに保存
    await writeFile(tempWebpFile, webpBuffer);
    console.log('WebP変換が完了しました');
    
    // 元のファイルとWebPのサイズを比較
    const originalSize = (await stat(tempInputFile)).size;
    const webpSize = webpBuffer.length;
    const compressionRatio = ((originalSize - webpSize) / originalSize * 100).toFixed(2);
    console.log(`圧縮率: ${compressionRatio}% (元: ${originalSize}バイト, WebP: ${webpSize}バイト)`);
    
    // CloudflareワーカーにWebP画像をアップロード
    console.log('変換されたWebP画像をR2にアップロードします');
    const formData = new FormData();
    formData.append('file', new Blob([webpBuffer]), webpOutputFileName);
    formData.append('type', 'webp');
    formData.append('fileName', webpOutputFileName);
    
    const uploadResponse = await axios.post(
      `${config.CLOUDFLARE_WORKER_URL}/upload`,
      formData,
      {
        headers: {
          'X-API-Key': config.API_KEY,
          'Content-Type': 'multipart/form-data'
        }
      }
    );
    
    // RDSのデータを更新
    await updateImageStatus(
      imageData.id, 
      'processed', 
      uploadResponse.data.path,
      null,
      {
        originalSize,
        webpSize,
        compressionRatio: parseFloat(compressionRatio),
        width: metadata.width,
        height: metadata.height
      },
      config
    );
    console.log('RDSのステータスを更新しました');
    
    // 一時ファイルを削除
    await Promise.all([
      unlink(tempInputFile),
      unlink(tempWebpFile)
    ]);
    console.log('一時ファイルを削除しました');
    
    // オリジナル画像をR2から削除
    await deleteFromR2(`original/${imageData.fileName}`, config);
    console.log('R2からオリジナル画像を削除しました');
    
    return {
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
  } catch (error) {
    console.error('画像処理中にエラーが発生しました:', error);
    // エラー時にRDSのステータスを更新
    await updateImageStatus(imageData.id, 'error', null, error.message, null, config);
    throw error;
  }
}

// バッチ処理関数
async function processBatch(batchSize = 10, config) {
  console.log(`最大${batchSize}件の画像を処理するバッチを開始します`);
  
  // RDSから処理待ちの画像を取得
  const pendingImages = await getImagesForProcessing(batchSize, config);
  console.log(`処理する画像が${pendingImages.length}件見つかりました`);
  
  if (pendingImages.length === 0) {
    return {
      status: 'success',
      message: '処理する画像がありません',
      processed: 0
    };
  }
  
  // 各画像を処理
  const results = [];
  for (const image of pendingImages) {
    try {
      // 処理中としてマーク
      await updateImageStatus(image.id, 'processing', null, null, null, config);
      
      // 画像処理
      const result = await processImage(image, config);
      results.push(result);
    } catch (error) {
      console.error(`画像ID ${image.id} の処理中にエラーが発生しました:`, error);
      results.push({
        status: 'error',
        imageId: image.id,
        message: error.message
      });
    }
  }
  
  return {
    status: 'success',
    message: 'バッチ処理が完了しました',
    processed: results.length,
    results: results
  };
}

// RDSから処理待ちの画像を取得 (mysql2使用)
async function getImagesForProcessing(limit, config) {
  console.log('処理待ちの画像をRDSから取得します');
  
  const connection = await getDbConnection(config);
  try {
    // RDSから処理待ちの画像を取得
    const [rows] = await connection.execute(
      `SELECT id, file_name as fileName, original_path as originalPath
       FROM images
       WHERE status = 'pending'
       LIMIT ?`,
      [limit]
    );
    
    return rows.map(row => ({
      id: row.id,
      fileName: row.fileName,
      originalPath: row.originalPath
    }));
  } catch (error) {
    console.error('RDSからの画像取得中にエラーが発生しました:', error);
    throw error;
  } finally {
    await connection.end();
  }
}

// 画像ステータスを更新 (mysql2使用)
async function updateImageStatus(imageId, status, webpPath = null, errorMessage = null, stats = null, config) {
  console.log(`画像ID ${imageId} のステータスを "${status}" に更新します`);
  
  const connection = await getDbConnection(config);
  try {
    let sqlStatement;
    let params;
    
    if (status === 'processed') {
      sqlStatement = `
        UPDATE images
        SET status = ?, 
            webp_path = ?, 
            original_size = ?,
            webp_size = ?,
            compression_ratio = ?,
            width = ?,
            height = ?,
            updated_at = NOW()
        WHERE id = ?
      `;
      params = [
        status,
        webpPath,
        stats ? stats.originalSize : 0,
        stats ? stats.webpSize : 0,
        stats ? stats.compressionRatio : 0,
        stats && stats.dimensions ? stats.dimensions.width : 0,
        stats && stats.dimensions ? stats.dimensions.height : 0,
        imageId
      ];
    } else if (status === 'error') {
      sqlStatement = `
        UPDATE images
        SET status = ?, error_message = ?, updated_at = NOW()
        WHERE id = ?
      `;
      params = [
        status,
        errorMessage || 'Unknown error',
        imageId
      ];
    } else {
      sqlStatement = `
        UPDATE images
        SET status = ?, updated_at = NOW()
        WHERE id = ?
      `;
      params = [
        status,
        imageId
      ];
    }
    
    await connection.execute(sqlStatement, params);
    return true;
  } catch (error) {
    console.error('RDSのステータス更新中にエラーが発生しました:', error);
    throw error;
  } finally {
    await connection.end();
  }
}

// R2からファイルを削除
async function deleteFromR2(path, config) {
  console.log(`R2からファイルを削除します: ${path}`);
  
  try {
    await axios({
      method: 'DELETE',
      url: `${config.CLOUDFLARE_WORKER_URL}/file/${path}`,
      headers: {
        'X-API-Key': config.API_KEY
      }
    });
    
    return true;
  } catch (error) {
    console.error('R2からのファイル削除中にエラーが発生しました:', error);
    throw error;
  }
}