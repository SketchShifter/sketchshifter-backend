// Cloudflare Worker for Image Upload to R2
// Simple implementation that handles only image uploads to R2

// KV Namespace binding: RATE_LIMITS (for rate limiting)
// R2 Bucket binding: SKETCHSHIFTER_BUCKET

// レート制限の設定
const RATE_LIMIT = {
  UPLOAD: {
    MAX: 20, // 1時間あたりの最大アップロード数
    WINDOW: 60 * 60 // 時間枠（秒）
  },
  API: {
    MAX: 100, // 1分あたりの最大API呼び出し数
    WINDOW: 60 // 時間枠（秒）
  }
};

// リクエストハンドラー
export default {
  async fetch(request, env, ctx) {
    try {
      // CORSヘッダーを設定
      const corsHeaders = {
        "Access-Control-Allow-Origin": "*",
        "Access-Control-Allow-Methods": "GET, PUT, POST, DELETE, OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type, Authorization, X-API-Key",
        "Access-Control-Max-Age": "86400"
      };

      // OPTIONSリクエストに対応
      if (request.method === "OPTIONS") {
        return new Response(null, {
          headers: corsHeaders,
          status: 204
        });
      }

      // URLを解析
      const url = new URL(request.url);
      const path = url.pathname;

      console.log(`リクエストを受信: ${request.method} ${path}`);

      // ヘルスチェックエンドポイント
      if (path === "/health") {
        return new Response(JSON.stringify({
          status: "ok",
          message: "Worker is running",
          timestamp: new Date().toISOString(),
          env: {
            hasKV: env.RATE_LIMITS != null,
            hasR2: env.SKETCHSHIFTER_BUCKET != null
          }
        }), { 
          headers: {
            ...corsHeaders,
            "Content-Type": "application/json"
          }
        });
      }

      // APIキー認証
      const apiKey = request.headers.get("X-API-Key");
      // if (!await validateApiKey(apiKey, env)) {
      //   console.log(`認証失敗: API Key=${apiKey}`);
      //   return new Response("Unauthorized: Invalid API Key", { 
      //     status: 401,
      //     headers: corsHeaders
      //   });
      // }

      // レート制限チェック
      const clientIP = request.headers.get("CF-Connecting-IP");
      if (!await checkRateLimit(clientIP, path.startsWith("/upload") ? "UPLOAD" : "API", env)) {
        return new Response("Rate limit exceeded", { 
          status: 429,
          headers: corsHeaders
        });
      }

      // パスに基づいて処理を分岐
      if (path === "/upload") {
        return handleUpload(request, env, corsHeaders);
      } else if (path.startsWith("/file/")) {
        // GETリクエストはファイル取得、DELETEリクエストはファイル削除
        if (request.method === "GET") {
          return handleFileAccess(request, env, corsHeaders, path);
        } else if (request.method === "DELETE") {
          return handleFileDelete(request, env, corsHeaders, path);
        } else {
          return new Response("Method not allowed", { 
            status: 405, 
            headers: corsHeaders 
          });
        }
      } else {
        console.log(`パスが見つかりません: ${path}`);
        return new Response("Not found", { 
          status: 404, 
          headers: corsHeaders 
        });
      }
    } catch (error) {
      console.error("Worker error:", error);
      return new Response("Internal error: " + error.message, { 
        status: 500,
        headers: {
          "Access-Control-Allow-Origin": "*",
          "Content-Type": "text/plain"
        }
      });
    }
  }
};

// ファイルアップロード処理
// ファイルアップロード処理でのデバッグ強化
async function handleUpload(request, env, corsHeaders) {
  if (request.method !== "POST") {
    return new Response("Method not allowed", { status: 405, headers: corsHeaders });
  }

  try {
    console.log("アップロード処理を開始");
    
    // リクエストからファイルとメタデータを取得
    const formData = await request.formData();
    console.log("FormData取得成功");
    
    const file = formData.get("file");
    console.log("Fileフィールド:", file ? "存在します" : "存在しません");
    
    if (!file) {
      console.log("ファイルがリクエストに含まれていません");
      return new Response(JSON.stringify({
        success: false,
        error: "File is required"
      }), { 
        status: 400, 
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }
    
    const fileName = formData.get("fileName") || generateFileName(file.name);
    console.log(`ファイル情報: fileName=${fileName}, type=${file.type}, size=${file.size}`);
    
    // ファイルタイプの確認（画像のみ許可）は一時的に無効化してテスト
    const contentType = file.type || "application/octet-stream";
    
    // R2にアップロード
    const path = `images/${fileName}`;
    console.log(`R2にアップロード開始: path=${path}`);
    
    await env.SKETCHSHIFTER_BUCKET.put(path, file, {
      httpMetadata: {
        contentType: contentType
      }
    });

    console.log(`アップロード成功: ${path}`);

    return new Response(JSON.stringify({
      success: true,
      path: path,
      url: `/file/${path}`
    }), {
      headers: {
        ...corsHeaders,
        "Content-Type": "application/json"
      }
    });
  } catch (error) {
    console.error("Upload error:", error.stack);
    return new Response(JSON.stringify({
      success: false, 
      error: "Upload failed: " + error.message,
      stack: error.stack
    }), {
      status: 500,
      headers: {
        ...corsHeaders,
        "Content-Type": "application/json"
      }
    });
  }
}
// ファイルアクセス処理
async function handleFileAccess(request, env, corsHeaders, path) {
  // パスからファイルパスを取得
  const filePath = path.replace(/^\/file\//, "");
  
  if (!filePath) {
    return new Response("Invalid file path", { status: 400, headers: corsHeaders });
  }

  console.log(`ファイルアクセス: ${filePath}`);

  // R2からオブジェクトを取得
  const object = await env.SKETCHSHIFTER_BUCKET.get(filePath);
  
  if (!object) {
    console.log(`ファイルが見つかりません: ${filePath}`);
    return new Response("File not found", { status: 404, headers: corsHeaders });
  }

  // レスポンスヘッダーの設定
  const headers = new Headers(corsHeaders);
  
  // Content-Typeヘッダーの設定
  if (object.httpMetadata?.contentType) {
    headers.set("Content-Type", object.httpMetadata.contentType);
  } else {
    // ファイル拡張子に基づいてContent-Typeを推定
    headers.set("Content-Type", getContentTypeFromExtension(filePath));
  }

  // キャッシュ制御ヘッダーの設定
  headers.set("Cache-Control", "public, max-age=31536000");

  console.log(`ファイル送信: ${filePath}, Content-Type: ${headers.get("Content-Type")}`);

  // ファイルを返す
  return new Response(object.body, { headers });
}

// ファイル削除処理
async function handleFileDelete(request, env, corsHeaders, path) {
  // パスからファイルパスを取得
  const filePath = path.replace(/^\/file\//, "");
  
  if (!filePath) {
    return new Response("Invalid file path", { 
      status: 400, 
      headers: corsHeaders 
    });
  }

  try {
    console.log(`ファイル削除: ${filePath}`);
    
    // R2からオブジェクトを削除
    await env.SKETCHSHIFTER_BUCKET.delete(filePath);
    
    return new Response(JSON.stringify({
      success: true,
      message: `File ${filePath} deleted successfully`
    }), {
      headers: {
        ...corsHeaders,
        "Content-Type": "application/json"
      }
    });
  } catch (error) {
    console.error("File deletion error:", error);
    
    return new Response(JSON.stringify({
      success: false,
      message: `Failed to delete file: ${error.message}`
    }), {
      status: 500,
      headers: {
        ...corsHeaders,
        "Content-Type": "application/json"
      }
    });
  }
}

// APIキーの検証関数
async function validateApiKey(apiKey, env) {
  // 開発段階では簡易的にハードコードされたキーを使用
  // const validKey = env.API_KEY || "your-api-key-here";
  return true;
}

// レート制限チェック関数
async function checkRateLimit(clientIP, actionType, env) {
  if (!clientIP) return true; // IPが取得できない場合は制限しない
  if (!env.RATE_LIMITS) return true; // KVがバインドされていない場合は制限しない
  
  const key = `${clientIP}:${actionType}`;
  const now = Math.floor(Date.now() / 1000);
  const limitConfig = RATE_LIMIT[actionType];
  
  if (!limitConfig) return true; // 設定がない場合は制限しない

  // KVから現在の使用状況を取得
  let usage = await env.RATE_LIMITS.get(key, { type: "json" }) || { count: 0, resetTime: now + limitConfig.WINDOW };
  
  // 時間枠が経過していればカウンターをリセット
  if (now > usage.resetTime) {
    usage = { count: 0, resetTime: now + limitConfig.WINDOW };
  }
  
  // 制限に達していればfalseを返す
  if (usage.count >= limitConfig.MAX) {
    return false;
  }
  
  // カウンターを更新
  usage.count++;
  await env.RATE_LIMITS.put(key, JSON.stringify(usage), { expirationTtl: limitConfig.WINDOW });
  
  return true;
}

// ファイル名を生成する関数
function generateFileName(originalName) {
  const timestamp = Date.now();
  const randomString = Math.random().toString(36).substring(2, 10);
  const extension = originalName.split('.').pop();
  return `${timestamp}_${randomString}.${extension}`;
}

// ファイルタイプが画像かどうかをチェック
function isImageContent(contentType) {
  return contentType && contentType.startsWith('image/');
}

// ファイル拡張子からContent-Typeを推定
function getContentTypeFromExtension(filePath) {
  const extension = filePath.split('.').pop().toLowerCase();
  const contentTypes = {
    'jpg': 'image/jpeg',
    'jpeg': 'image/jpeg',
    'png': 'image/png',
    'gif': 'image/gif',
    'webp': 'image/webp',
    'svg': 'image/svg+xml',
    'ico': 'image/x-icon'
  };
  
  return contentTypes[extension] || 'application/octet-stream';
}