import time

import schedule
from telegram import Bot
from config import CHAT_ID, TOKEN

bot = Bot(token=TOKEN)


def send_night_message():
    chat_id = CHAT_ID
    bot.send_message(chat_id=chat_id, text="Спокойной ночи, Гномы.")


def schedule_jobs():
    schedule.every().day.at("23:00").do(send_night_message())

    while True:
        schedule.run_pending()
        time.sleep(1)


if __name__ == "__main__":
    schedule_jobs()
