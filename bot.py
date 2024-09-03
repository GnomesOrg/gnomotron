from config import TOKEN
from telegram.ext import Application, CommandHandler, filters, CallbackContext, MessageHandler, ChatMemberHandler
from handlers import start, help_command, echo, handle_photo
import asyncio


def main():
    application = Application.builder().token(TOKEN).build()

    application.add_handler(CommandHandler("start", start))
    application.add_handler(CommandHandler("help", help_command))

    application.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, echo))
    application.add_handler(MessageHandler(filters.PHOTO, handle_photo))

    application.run_polling()

    # try:
    #     await application.initialize()
    #
    #     # code
    # finally:
    #     await application.shutdown()


if __name__ == "__main__":
    main()
