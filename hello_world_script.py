def wsgi(environ, start_response):
    status = '200 OK'
    headers = [('Content-type', 'text/html')]
    start_response(status, headers)
    
    html_content = '''
    <!DOCTYPE html>
    <html>
    <head>
        <title>Alice's Dyson Protocol DWapp</title>
        <style>
            body {
                font-family: Arial, sans-serif;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                margin: 0;
                padding: 40px;
                color: white;
                text-align: center;
                min-height: 100vh;
                display: flex;
                flex-direction: column;
                justify-content: center;
            }
            .container {
                max-width: 600px;
                margin: 0 auto;
                background: rgba(255, 255, 255, 0.1);
                padding: 40px;
                border-radius: 20px;
                backdrop-filter: blur(10px);
                box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
            }
            h1 {
                font-size: 3em;
                margin-bottom: 20px;
                text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.3);
            }
            p {
                font-size: 1.2em;
                line-height: 1.6;
                margin-bottom: 15px;
            }
            .address {
                font-family: monospace;
                background: rgba(0, 0, 0, 0.2);
                padding: 10px;
                border-radius: 8px;
                word-break: break-all;
                font-size: 0.9em;
            }
            .emoji {
                font-size: 2em;
                margin: 20px 0;
            }
        </style>
    </head>
    <body>
        <div class="container">
            <h1>🚀 Hello from Alice!</h1>
            <div class="emoji">👋 🌟 ✨</div>
            <p>Welcome to Alice's decentralized web application powered by Dyson Protocol!</p>
            <p>This page is served directly from the blockchain using the name <strong>alice.dys</strong></p>
            <p>Alice's address:</p>
            <div class="address">dys1prvefcdgdnh2cpas6rnnval84n0gv2r28tklr4</div>
            <p>🎉 Decentralized web hosting is now live! 🎉</p>
        </div>
    </body>
    </html>
    '''
    
    return [html_content.encode('utf-8')]

def hello():
    """A simple function that returns a greeting"""
    return {"message": "Hello from Alice's script!", "status": "success"} 