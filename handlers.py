import random

from telegram.ext import Updater, CommandHandler, CallbackContext
from telegram import Update


async def start(update: Update, context: CallbackContext) -> None:
    await update.message.reply_text('Привет! Я гномотрон')


async def help_command(update: Update, context: CallbackContext) -> None:
    chat_id = update.effective_chat.id

    await update.message.reply_text('Current chat id is: ' + str(chat_id))


async def echo(update: Update, context: CallbackContext) -> None:
    if should_reply():
        await update.message.reply_text(update.message.text + " - сказал пьяница")


async def handle_photo(update: Update, context: CallbackContext) -> None:
    await update.message.reply_text('Красивое фото пьяницы')


def should_reply(probability=0.05) -> bool:
    return random.random() < probability
