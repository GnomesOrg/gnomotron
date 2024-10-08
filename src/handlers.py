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
    message_text = update.message.text
    
    if len(message_text) >= 40 and should_reply():
        await update.message.reply_text(get_gpt_response_with_message(message_text))



async def handle_photo(update: Update, context: CallbackContext) -> None:
    if should_reply(0.3):
        reactions = [
            "Тебе показали забавную картинку. Опиши свою реакцию",
            "Перед тобой милое изображение. Как ты отреагируешь?",
            "На экране смешная фотография. Какой твой ответ?",
            "Ты видишь что-то необычное. Как бы ты это описал?",
            "Картинка выглядит странно. Что ты скажешь?",
            "Это что-то очень милое! Какова твоя реакция?"
        ]

        random_reaction = random.choice(reactions)

        await update.message.reply_text(get_gpt_response_with_message(random_reaction))


def should_reply(probability=0.04) -> bool:
    return random.random() < probability
