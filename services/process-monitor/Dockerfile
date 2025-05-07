FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
ENV TUNNEL_AGENT_URL=http://tunnel-agent:8080 \
    SCAN_INTERVAL=2.5 \
    LOG_LEVEL=info
CMD ["python", "monitor.py"]
