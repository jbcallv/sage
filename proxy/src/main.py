import httpx
from fastapi import FastAPI, Request
from fastapi.responses import Response
from urllib.parse import unquote

app = FastAPI()
client = httpx.AsyncClient()


@app.get("/api/health")
async def health():
    return {"status": "ok"}


@app.middleware("http")
async def proxy_middleware(request: Request, call_next):
    path = request.scope.get("path", "")

    if path.startswith("/api"):
        return await call_next(request)

    # explicit proxy mode: client sent full URL as path
    if path.startswith("http"):
        target = unquote(path)
    else:
        # transparent mode: reconstruct from Host header + path
        host = request.headers.get("host", "")
        query = f"?{request.url.query}" if request.url.query else ""
        target = f"http://{host}{path}{query}"

    headers = {k: v for k, v in request.headers.items() if k.lower() != "host"}

    resp = await client.request(
        method=request.method,
        url=target,
        headers=headers,
        content=await request.body(),
    )

    return Response(
        content=resp.content,
        status_code=resp.status_code,
        headers=dict(resp.headers),
    )
