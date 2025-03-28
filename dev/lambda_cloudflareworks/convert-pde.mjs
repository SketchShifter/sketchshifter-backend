import { SQSClient, SendMessageCommand } from '@aws-sdk/client-sqs';
import { SSMClient, GetParameterCommand } from '@aws-sdk/client-ssm';
import mysql from 'mysql2/promise';
import axios from 'axios';
import fs from 'fs';
import path from 'path';
import { promisify } from 'util';

const writeFile = promisify(fs.writeFile);
const readFile = promisify(fs.readFile);
const unlink = promisify(fs.unlink);

// AWS クライアントの初期化
const sqs = new SQSClient({ region: process.env.AWS_REGION || 'ap-northeast-1' });
const ssm = new SSMClient({ region: process.env.AWS_REGION || 'ap-northeast-1' });

// キャッシュされたパラメータ
let cachedParams = null;
let paramsLastFetchTime = 0;
const PARAM_CACHE_TTL = 15 * 60 * 1000; // 15分

// Systems Managerパラメータストアから設定を取得
async function getParameters() {
  // キャッシュが有効なら使用
  const now = Date.now();
  if (cachedParams && now - paramsLastFetchTime < PARAM_CACHE_TTL) {
    return cachedParams;
  }

  try {
    // 環境変数から直接DB情報を取得できるか確認
    if (process.env.RDS_HOST && process.env.RDS_USER && process.env.RDS_PASSWORD) {
      cachedParams = {
        CLOUDFLARE_WORKER_URL: process.env.CLOUDFLARE_WORKER_URL,
        API_KEY: process.env.API_KEY,
        RDS_HOST: process.env.RDS_HOST,
        RDS_PORT: process.env.RDS_PORT || '3306',
        RDS_USER: process.env.RDS_USER,
        RDS_PASSWORD: process.env.RDS_PASSWORD,
        RDS_DATABASE: process.env.RDS_DATABASE
      };
      
      paramsLastFetchTime = now;
      return cachedParams;
    }
    
    // パラメータパスをカスタマイズ
    const paramPath = process.env.PARAM_PATH || '/sketchshifter/config';
    
    const command = new GetParameterCommand({
      Name: paramPath,
      WithDecryption: true
    });
    
    const response = await ssm.send(command);
    
    // パラメータ値をJSONとして解析
    const config = JSON.parse(response.Parameter.Value);
    
    // キャッシュを更新
    cachedParams = {
      CLOUDFLARE_WORKER_URL: config.cloudflareWorkerUrl,
      API_KEY: config.apiKey,
      RDS_HOST: config.rdsHost,
      RDS_PORT: config.rdsPort || '3306',
      RDS_USER: config.rdsUser,
      RDS_PASSWORD: config.rdsPassword,
      RDS_DATABASE: config.rdsDatabase
    };
    
    paramsLastFetchTime = now;
    return cachedParams;
  } catch (error) {
    console.error('パラメータの取得中にエラーが発生しました:', error);
    
    // エラーが発生した場合は環境変数を使用
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

// データベース接続を取得
async function getDbConnection(config) {
  return await mysql.createConnection({
    host: config.RDS_HOST,
    port: parseInt(config.RDS_PORT, 10),
    user: config.RDS_USER,
    password: config.RDS_PASSWORD,
    database: config.RDS_DATABASE,
    // 接続プールの設定（オプション）
    waitForConnections: true,
    connectionLimit: 10,
    queueLimit: 0
  });
}

export const handler = async (event) => {
  console.log('PDE→JS変換処理を開始します');
  
  // 設定を取得
  const config = await getParameters();
  
  // SQSからのイベントを処理
  if (event.Records && event.Records.length > 0) {
    const results = [];
    
    for (const record of event.Records) {
      try {
        const messageBody = JSON.parse(record.body);
        console.log('処理するメッセージ:', messageBody);
        
        // PDEファイルの処理
        const result = await processFile(messageBody.processingData, config);
        results.push(result);
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
  } else if (event.processingData) {
    // 直接のLambda呼び出しからの処理
    try {
      const result = await processFile(event.processingData, config);
      return {
        statusCode: 200,
        body: JSON.stringify(result)
      };
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
  } else {
    return {
      statusCode: 400,
      body: JSON.stringify({
        status: 'error',
        message: 'SQSメッセージまたは処理データが必要です'
      })
    };
  }
};

// PDEファイルの処理
async function processFile(processingData, config) {
  console.log(`PDEファイルを処理します: ${processingData.fileName}`);
  
  try {
    // 一時ファイル名の設定
    const tempInputFile = `/tmp/${processingData.fileName}`;
    const outputFileName = processingData.fileName.replace(/\.pde$/, '.js');
    const tempOutputFile = `/tmp/${outputFileName}`;
    
    // PDEファイルの内容を取得
    let pdeContent;
    
    if (processingData.pdeContent) {
      // PDEテキストが直接渡された場合
      pdeContent = processingData.pdeContent;
      console.log('PDEコンテンツが直接提供されました');
      
      // 一時ファイルに保存
      await writeFile(tempInputFile, pdeContent);
    } else if (processingData.pdePath) {
      // R2からPDEファイルを取得
      console.log('CloudflareワーカーからR2のPDEファイルを取得します');
      const r2Response = await axios({
        method: 'GET',
        url: `${config.CLOUDFLARE_WORKER_URL}/file/${processingData.pdePath}`,
        responseType: 'text',
        headers: {
          'X-API-Key': config.API_KEY
        }
      });
      
      pdeContent = r2Response.data;
      
      // PDEファイルを一時ファイルに保存
      await writeFile(tempInputFile, pdeContent);
      console.log('PDEファイルを一時ファイルに保存しました');
    } else {
      // RDSからPDEコンテンツを取得
      pdeContent = await getPdeContentFromDatabase(processingData.id, config);
      
      if (!pdeContent) {
        throw new Error('PDEコンテンツが見つかりません');
      }
      
      // PDEファイルを一時ファイルに保存
      await writeFile(tempInputFile, pdeContent);
      console.log('PDEファイルをデータベースから取得して一時ファイルに保存しました');
    }
    
    // PDE → JS変換処理
    console.log('PDEをJavaScriptに変換します');
    const jsCode = await convertPdeToJs(tempInputFile, processingData);
    
    // 変換されたJSファイルを保存
    await writeFile(tempOutputFile, jsCode);
    console.log('変換されたJSファイルを保存しました');
    
    // CloudflareワーカーにJSファイルをアップロード
    console.log('変換されたJSファイルをR2にアップロードします');
    const jsData = await readFile(tempOutputFile, 'utf8');
    
    const formData = new FormData();
    formData.append('file', new Blob([jsData]), outputFileName);
    formData.append('type', 'js');
    formData.append('fileName', outputFileName);
    
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
    await updateProcessingStatus(
      processingData.id, 
      'processed', 
      uploadResponse.data.path, 
      null, 
      config
    );
    console.log('RDSのステータスを更新しました');
    
    // 一時ファイルを削除
    await Promise.all([
      unlink(tempInputFile),
      unlink(tempOutputFile)
    ]);
    console.log('一時ファイルを削除しました');
    
    return {
      status: 'success',
      processingId: processingData.id,
      jsPath: uploadResponse.data.path
    };
  } catch (error) {
    console.error('PDE処理中にエラーが発生しました:', error);
    // エラー時にRDSのステータスを更新
    await updateProcessingStatus(processingData.id, 'error', null, error.message, config);
    throw error;
  }
}

// データベースからPDEコンテンツを取得
async function getPdeContentFromDatabase(processingId, config) {
  console.log(`処理ID ${processingId} のPDEコンテンツをデータベースから取得します`);
  
  const connection = await getDbConnection(config);
  try {
    const [rows] = await connection.execute(
      `SELECT pde_content FROM processing_works WHERE id = ?`,
      [processingId]
    );
    
    if (rows.length === 0) {
      return null;
    }
    
    return rows[0].pde_content;
  } finally {
    await connection.end();
  }
}

// PDE → JS変換用の一時的な実装
// 実際の実装ではProcessing変換ライブラリを使用します
async function convertPdeToJs(pdeFilePath, processingData) {
  // PDEファイルの内容を読み込み
  const pdeContent = await readFile(pdeFilePath, 'utf8');
  
  // この関数は実際のProcessing変換ロジックに置き換える必要があります
  // ここでは簡易的なラッパーを作成するだけ
  
  // Processing.jsの基本的なラッパーを生成
  const jsCode = `
    // Auto-generated from ${processingData.fileName}
    // Original PDE file: ${processingData.originalName || processingData.fileName}
    
    (function() {
      // Processing.jsのセットアップ
      var canvas = document.getElementById('${processingData.canvasId || "processingCanvas"}');
      var processing = new Processing(canvas, function(p) {
        // Processing変数の設定
        var width = canvas.width;
        var height = canvas.height;
        
        // オリジナルのPDEコード（Javascriptに変換された想定）
        ${convertPdeToJsContent(pdeContent)}
        
        // Processing実行時のメソッド呼び出し
        p.setup = setup || function() {};
        p.draw = draw || function() {};
        p.mousePressed = mousePressed || function() {};
        p.mouseReleased = mouseReleased || function() {};
        p.keyPressed = keyPressed || function() {};
        p.keyReleased = keyReleased || function() {};
      });
    })();
  `;
  
  return jsCode;
}

// PDEコードをJS形式に変換する仮実装
// 実際には専用のトランスパイラを使用します
function convertPdeToJsContent(pdeCode) {
  // この関数は実際の変換ロジックに置き換えてください
  // 現在は単にコメントを追加するだけの仮実装
  
  let jsCode = '// PDE code converted to JS\n';
  
  // 簡単な置換を行う
  const converted = pdeCode
    // void setup() → function setup()
    .replace(/void\s+setup\s*\(\s*\)\s*\{/g, 'function setup() {')
    // void draw() → function draw()
    .replace(/void\s+draw\s*\(\s*\)\s*\{/g, 'function draw() {')
    // その他のProcessing固有の関数変換
    .replace(/void\s+(\w+)\s*\(/g, 'function $1(')
    // サイズ設定
    .replace(/size\s*\(\s*(\d+)\s*,\s*(\d+)\s*\)/g, 'p.size($1, $2)')
    // 基本的な描画関数
    .replace(/background\s*\(/g, 'p.background(')
    .replace(/fill\s*\(/g, 'p.fill(')
    .replace(/stroke\s*\(/g, 'p.stroke(')
    .replace(/rect\s*\(/g, 'p.rect(')
    .replace(/ellipse\s*\(/g, 'p.ellipse(')
    .replace(/line\s*\(/g, 'p.line(')
    .replace(/point\s*\(/g, 'p.point(')
    // その他の関数
    .replace(/frameRate\s*\(/g, 'p.frameRate(');
  
  jsCode += converted;
  return jsCode;
}

// 処理ステータスを更新
async function updateProcessingStatus(processingId, status, jsPath = null, errorMessage = null, config) {
  console.log(`処理ID ${processingId} のステータスを "${status}" に更新します`);
  
  let sqlStatement;
  let queryParams;
  
  if (status === 'processed') {
    sqlStatement = `
      UPDATE processing_works
      SET status = ?, js_path = ?, updated_at = NOW()
      WHERE id = ?
    `;
    queryParams = [status, jsPath, processingId];
  } else if (status === 'error') {
    sqlStatement = `
      UPDATE processing_works
      SET status = ?, error_message = ?, updated_at = NOW()
      WHERE id = ?
    `;
    queryParams = [status, errorMessage || 'Unknown error', processingId];
  } else {
    sqlStatement = `
      UPDATE processing_works
      SET status = ?, updated_at = NOW()
      WHERE id = ?
    `;
    queryParams = [status, processingId];
  }
  
  const connection = await getDbConnection(config);
  try {
    await connection.execute(sqlStatement, queryParams);
    return true;
  } catch (error) {
    console.error('RDSのステータス更新中にエラーが発生しました:', error);
    throw error;
  } finally {
    await connection.end();
  }
}