import asyncio
import time

import schedule
from telegram import Bot
from config import CHAT_ID, TOKEN

bot = Bot(token=TOKEN)


async def send_simple_message(message):
    chat_id = CHAT_ID
    print(123)
    await bot.send_message(chat_id=chat_id, text=message)

def schedule_jobs():
    """Night message"""
    schedule.every().day.at("23:00").do(lambda: asyncio.run(send_simple_message("Спокойной ночи, Гномы.")))
    # timestr = "15:54"
    # schedule.every().day.at(timestr).do(lambda: asyncio.run(send_simple_message("Тестовое уведомление в " + timestr)))

    while True:
        schedule.run_pending()
        time.sleep(1)
