FROM python:3.10-alpine

COPY ./requirements.txt /app/

RUN pip install -r /app/requirements.txt --no-cache-dir

COPY ./src /app/src

EXPOSE 8080

ENTRYPOINT python3 /app/src/bot.py
