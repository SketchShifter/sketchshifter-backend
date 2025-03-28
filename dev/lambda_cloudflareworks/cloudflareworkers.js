// Cloudflare Worker for SketchShifter Platform
// R2アクセスと署名付きURL生成を管理

// KV Namespace binding: RATE_LIMITS
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
  
  // URL署名の設定
  const URL_SIGNING = {
    EXPIRY: 10 * 60, // 署名付きURLの有効期間（秒）
    SECRET: "change-this-to-a-secure-random-string"
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
        if (!await validateApiKey(apiKey, env)) {
          console.log(`認証失敗: API Key=${apiKey}`);
          return new Response("Unauthorized: Invalid API Key", { 
            status: 401,
            headers: corsHeaders
          });
        }
  
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
        } else if (path === "/get-signed-url") {
          return handleSignedUrlGeneration(request, env, corsHeaders);
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
  async function handleUpload(request, env, corsHeaders) {
    if (request.method !== "POST") {
      return new Response("Method not allowed", { status: 405, headers: corsHeaders });
    }
  
    try {
      console.log("アップロード処理を開始");
      
      // リクエストからファイルとメタデータを取得
      const formData = await request.formData();
      const file = formData.get("file");
      const type = formData.get("type") || "original"; // original, pde, webp, js
      const fileName = formData.get("fileName") || crypto.randomUUID();
      
      console.log(`ファイル情報: type=${type}, fileName=${fileName}`);
      
      if (!file) {
        return new Response("File is required", { status: 400, headers: corsHeaders });
      }
  
      // ファイルタイプに応じたパスを決定
      let path;
      if (type === "pde") {
        path = `pde/${fileName}`;
      } else if (type === "webp") {
        path = `webp/${fileName}`;
      } else if (type === "js") {
        path = `js/${fileName}`;
      } else {
        path = `original/${fileName}`;
      }
  
      // ファイルタイプの確認
      let contentType;
      if (type === "webp") {
        contentType = "image/webp";
      } else if (type === "js") {
        contentType = "application/javascript";
      } else if (type === "pde") {
        contentType = "text/plain";
      } else {
        // オリジナル画像の場合はファイル名から推測
        if (fileName.endsWith(".jpg") || fileName.endsWith(".jpeg")) {
          contentType = "image/jpeg";
        } else if (fileName.endsWith(".png")) {
          contentType = "image/png";
        } else if (fileName.endsWith(".gif")) {
          contentType = "image/gif";
        } else if (fileName.endsWith(".webp")) {
          contentType = "image/webp";
        } else if (fileName.endsWith(".pde")) {
          contentType = "text/plain";
        } else {
          contentType = "application/octet-stream";
        }
      }
  
      console.log(`R2にアップロード: path=${path}, contentType=${contentType}`);
  
      // R2にアップロード（カスタムメタデータ付き）
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
      console.error("Upload error:", error);
      return new Response("Upload failed: " + error.message, {
        status: 500,
        headers: corsHeaders
      });
    }
  }
  
  // 署名付きURL生成処理
  async function handleSignedUrlGeneration(request, env, corsHeaders) {
    if (request.method !== "POST") {
      return new Response("Method not allowed", { status: 405, headers: corsHeaders });
    }
  
    try {
      const { type, contentType, fileName } = await request.json();
      
      if (!type || !fileName) {
        return new Response("Type and fileName are required", {
          status: 400,
          headers: corsHeaders
        });
      }
  
      // ファイルタイプに応じたパスを決定
      let path;
      if (type === "pde") {
        path = `pde/${fileName}`;
      } else if (type === "webp") {
        path = `webp/${fileName}`;
      } else if (type === "js") {
        path = `js/${fileName}`;
      } else {
        path = `original/${fileName}`;
      }
  
      // 署名付きURLを生成
      const expiryTime = Math.floor(Date.now() / 1000) + URL_SIGNING.EXPIRY;
      const signature = await generateSignature(path, expiryTime, env);
      
      const signedUrl = new URL(request.url);
      signedUrl.pathname = "/upload";
      signedUrl.searchParams.set("path", path);
      signedUrl.searchParams.set("expires", expiryTime.toString());
      signedUrl.searchParams.set("signature", signature);
  
      return new Response(JSON.stringify({
        url: signedUrl.toString(),
        expires: expiryTime,
        method: "PUT"
      }), {
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    } catch (error) {
      console.error("Signed URL generation error:", error);
      return new Response("Failed to generate signed URL: " + error.message, {
        status: 500,
        headers: corsHeaders
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
      if (filePath.endsWith(".webp")) {
        headers.set("Content-Type", "image/webp");
      } else if (filePath.endsWith(".jpg") || filePath.endsWith(".jpeg")) {
        headers.set("Content-Type", "image/jpeg");
      } else if (filePath.endsWith(".png")) {
        headers.set("Content-Type", "image/png");
      } else if (filePath.endsWith(".pde")) {
        headers.set("Content-Type", "text/plain");
      } else if (filePath.endsWith(".js")) {
        headers.set("Content-Type", "application/javascript");
      }
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
  
  // 署名を生成する関数
  async function generateSignature(path, expires, env) {
    const encoder = new TextEncoder();
    const data = encoder.encode(`${path}:${expires}:${URL_SIGNING.SECRET}`);
    const hashBuffer = await crypto.subtle.digest("SHA-256", data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, "0")).join("");
  }
  
  // APIキーの検証関数
  async function validateApiKey(apiKey, env) {
    // 開発段階では簡易的にすべてのキーを許可
    return true;
    
    // 実際の実装（後で有効化）
    // return apiKey === "YOUR_API_KEY_HERE" || apiKey === env.API_KEY;
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