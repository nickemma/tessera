package httpapi

import (
	"net/http"
	"strings"
)

func openAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(openAPISpec))
}

func playground(demoKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(strings.ReplaceAll(playgroundHTML, "__DEMO_KEY__", demoKey)))
	}
}

const openAPISpec = `{
  "openapi": "3.0.3",
  "info": {"title": "Tessera Gateway", "version": "0.1.0", "description": "Local playground API for the Tessera inference platform."},
  "servers": [{"url": "http://localhost:8080"}],
  "paths": {
    "/healthz": {"get": {"summary": "Health check", "responses": {"200": {"description": "Healthy"}}}},
    "/v1/chat/completions": {
      "post": {
        "summary": "Create a chat completion",
        "security": [{"bearerAuth": []}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/ChatRequest"}}}},
        "responses": {"200": {"description": "Completion returned"}, "401": {"description": "Invalid API key"}, "429": {"description": "Budget exceeded"}}
      }
    },
    "/v1/completions": {
      "post": {"summary": "Create a text completion", "security": [{"bearerAuth": []}], "responses": {"200": {"description": "Completion returned"}, "401": {"description": "Invalid API key"}, "429": {"description": "Budget or rate limit exceeded"}}}
    },
    "/v1/usage": {"get": {"summary": "Get tenant usage", "security": [{"bearerAuth": []}], "responses": {"200": {"description": "Usage summary"}}}}
  },
  "components": {
    "securitySchemes": {"bearerAuth": {"type": "http", "scheme": "bearer"}},
    "schemas": {"ChatRequest": {"type": "object", "required": ["messages"], "properties": {"model": {"type": "string", "example": "canned-local"}, "messages": {"type": "array", "items": {"type": "object", "required": ["role", "content"], "properties": {"role": {"type": "string"}, "content": {"type": "string"}}}}, "max_tokens": {"type": "integer", "minimum": 1}, "stream": {"type": "boolean"}}}}
  }
}`

const playgroundHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Tessera Playground</title>
  <style>
    :root{color-scheme:dark;font-family:system-ui,sans-serif}body{max-width:900px;margin:40px auto;padding:0 20px;background:#111827;color:#e5e7eb}input,textarea,button{font:inherit;border:1px solid #374151;border-radius:6px;padding:10px;background:#1f2937;color:#f9fafb}input,textarea{width:100%;box-sizing:border-box;margin:6px 0 16px}textarea{min-height:130px}button{background:#2563eb;border:0;cursor:pointer}button:disabled{opacity:.6}pre{white-space:pre-wrap;background:#030712;border:1px solid #374151;border-radius:6px;padding:16px;min-height:120px}.hint{color:#9ca3af;font-size:.9rem}a{color:#93c5fd}
  </style>
</head>
<body>
  <h1>Tessera Playground</h1>
  <p class="hint">This is the local v0 playground. The demo key is seeded in memory and is not for production use.</p>
  <label>API key<input id="key" value="__DEMO_KEY__"></label>
  <label>Model<input id="model" value="canned-local"></label>
  <label>Prompt<textarea id="prompt">Explain what Tessera does in one sentence.</textarea></label>
  <label><input id="stream" type="checkbox" style="width:auto"> Stream response</label>
  <button id="send">Send request</button>
  <p><a href="/openapi.json" target="_blank">View OpenAPI specification</a></p>
  <pre id="output">Response will appear here.</pre>
  <script>
    const output=document.getElementById('output');
    document.getElementById('send').onclick=async()=>{
      const button=document.getElementById('send'); button.disabled=true; output.textContent='Loading...';
      try{
        const response=await fetch('/v1/chat/completions',{method:'POST',headers:{'Authorization':'Bearer '+document.getElementById('key').value,'Content-Type':'application/json'},body:JSON.stringify({model:document.getElementById('model').value,messages:[{role:'user',content:document.getElementById('prompt').value}],stream:document.getElementById('stream').checked})});
        output.textContent=await response.text();
      }catch(error){output.textContent=String(error)}finally{button.disabled=false}
    };
  </script>
</body>
</html>`
