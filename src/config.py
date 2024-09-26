import os

TOKEN = os.environ["GNOMOTRON_TELEGRAM_TOKEN"]
CHAT_ID = os.environ["GNOMOTRON_TELEGRAM_CHAT_ID"]
API_KEY = os.environ["GNOMOTRON_TELEGRAM_API_KEY"]
SHOULD_DROP_MESSAGES = True

if not TOKEN or not CHAT_ID or not API_KEY:
    raise RuntimeError("TOKEN, CHAT_ID and API_KEY must be specified via the env")
