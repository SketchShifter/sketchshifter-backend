// AWS Lambda 関数 - PDE to JavaScript 変換
const axios = require('axios');

exports.handler = async (event) => {
    console.log('PDE変換処理を開始します');
    console.log('受信イベント:', JSON.stringify(event));
    
    try {
        // リクエストからデータを取得
        const body = typeof event.body === 'string' ? JSON.parse(event.body) : event.body;
        
        // 必須パラメータのチェック
        if (!body || !body.processingId || !body.pdeContent) {
            return {
                statusCode: 400,
                body: JSON.stringify({
                    success: false,
                    message: 'Missing required parameters: processingId and pdeContent'
                })
            };
        }
        
        try {
            // PDEをJavaScriptに変換
            const jsContent = convertPdeToJs(body.pdeContent, body.canvasId || 'processingCanvas', body.fileName || 'sketch.pde');

            // 直接結果を返す
            return {
                statusCode: 200,
                body: JSON.stringify({
                    success: true,
                    processingId: body.processingId,
                    jsContent: jsContent
                })
            };
        } catch (error) {
            console.error('変換処理に失敗しました:', error);
            return {
                statusCode: 500,
                body: JSON.stringify({
                    success: false,
                    processingId: body.processingId,
                    message: `変換処理に失敗しました: ${error.message}`
                })
            };
        }
    } catch (error) {
        console.error('PDE変換処理に失敗しました:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({
                success: false,
                message: `PDE変換処理に失敗しました: ${error.message}`
            })
        };
    }
};

// PDEコードをJS形式に変換する関数
function convertPdeToJs(pdeCode, canvasId = 'processingCanvas', fileName = 'sketch.pde') {
    if (!pdeCode) {
        throw new Error('PDEコンテンツが空です');
    }
    
    // コメントを追加
    let jsCode = '// PDE code converted to JS\n';
    
    // 簡単な置換を行う
    const convertedCode = pdeCode
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
        .replace(/frameRate\s*\(/g, 'p.frameRate(')
        // イベント処理
        .replace(/mouseX/g, 'p.mouseX')
        .replace(/mouseY/g, 'p.mouseY')
        .replace(/mousePressed/g, 'p.mousePressed')
        .replace(/keyPressed/g, 'p.keyPressed')
        // 変数と定数
        .replace(/PI/g, 'Math.PI')
        .replace(/HALF_PI/g, 'Math.PI / 2')
        .replace(/QUARTER_PI/g, 'Math.PI / 4')
        .replace(/TWO_PI/g, 'Math.PI * 2')
        // 数学関数
        .replace(/abs\s*\(/g, 'Math.abs(')
        .replace(/sqrt\s*\(/g, 'Math.sqrt(')
        .replace(/sin\s*\(/g, 'Math.sin(')
        .replace(/cos\s*\(/g, 'Math.cos(')
        .replace(/tan\s*\(/g, 'Math.tan(')
        .replace(/random\s*\(/g, 'p.random(');
    
    jsCode += convertedCode;
    
    // Processing.jsの基本的なラッパーを生成
    const wrapperCode = `
// Auto-generated from ${fileName} at ${new Date().toISOString()}
// Processing.js wrapper

(function() {
  // Processing.jsのセットアップ
  var canvas = document.getElementById('${canvasId}');
  if (!canvas) {
    console.error('Canvas element not found: ${canvasId}');
    return;
  }
  
  var processing = new Processing(canvas, function(p) {
    // Processing変数の設定
    var width = canvas.width;
    var height = canvas.height;
    
    // オリジナルのPDEコード（Javascriptに変換）
    ${jsCode}
    
    // Processing実行時のメソッド呼び出し
    p.setup = typeof setup !== 'undefined' ? setup : function() {};
    p.draw = typeof draw !== 'undefined' ? draw : function() {};
    p.mousePressed = typeof mousePressed !== 'undefined' ? mousePressed : function() {};
    p.mouseReleased = typeof mouseReleased !== 'undefined' ? mouseReleased : function() {};
    p.keyPressed = typeof keyPressed !== 'undefined' ? keyPressed : function() {};
    p.keyReleased = typeof keyReleased !== 'undefined' ? keyReleased : function() {};
  });
})();`;

    return wrapperCode;
}