import threading
from config import TOKEN, SHOULD_DROP_MESSAGES
from telegram.ext import Application, CommandHandler, filters, CallbackContext, MessageHandler, ChatMemberHandler
from handlers import start, help_command, echo, handle_photo
from scheduler import schedule_jobs


def main():
    schedule_thread = threading.Thread(target=schedule_jobs)
    schedule_thread.start()

    application = Application.builder().token(TOKEN).drop_pending_updates(SHOULD_DROP_MESSAGES).build()

    application.add_handler(CommandHandler("start", start))
    application.add_handler(CommandHandler("help", help_command))

    application.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, echo))
    application.add_handler(MessageHandler(filters.PHOTO, handle_photo))

    application.run_polling()


if __name__ == "__main__":
    main()
