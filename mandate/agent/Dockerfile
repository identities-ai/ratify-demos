FROM python:3.12-slim

WORKDIR /app

COPY requirements-smoke.txt requirements.txt ./
RUN pip install --no-cache-dir -r requirements-smoke.txt

COPY . .

ENV CLOUDOPS_MCP_URL=http://cloudops-mcp:8090/mcp
ENV VENDOR_MCP_URL=http://vendor-mcp:8091/mcp

CMD ["python", "smoke.py"]
