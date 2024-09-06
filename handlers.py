import random

from telegram.ext import Updater, CommandHandler, CallbackContext
from telegram import Update

from gpt_adapter import get_gpt_response_with_message


async def start(update: Update, context: CallbackContext) -> None:
    await update.message.reply_text('Привет! Я гномотрон')


async def help_command(update: Update, context: CallbackContext) -> None:
    chat_id = update.effective_chat.id

    await update.message.reply_text('Current chat id is: ' + str(chat_id))


async def help_gpt(update: Update, context: CallbackContext) -> None:
    text = update.message.text.split(" ", 1)[1]
    await update.message.reply_text(get_gpt_response_with_message(text))


async def echo(update: Update, context: CallbackContext) -> None:
    if should_reply():
        await update.message.reply_text(get_gpt_response_with_message(update.message.text))


async def handle_photo(update: Update, context: CallbackContext) -> None:
    if should_reply(0.5):
        await update.message.reply_text(random.choice(["Красивое фото пьяницы", "Смешной прикол!!", "Удали."]))


def should_reply(probability=0.25) -> bool:
    return random.random() < probability
