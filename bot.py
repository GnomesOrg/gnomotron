import threading

from scheduler import schedule_jobs
from config import TOKEN, SHOULD_DROP_MESSAGES
from telegram.ext import Application, CommandHandler, filters, CallbackContext, MessageHandler, ChatMemberHandler
from handlers import start, help_command, echo, handle_photo


def main():
    application = Application.builder().token(TOKEN).build()

    application.add_handler(CommandHandler("start", start))
    application.add_handler(CommandHandler("help", help_command))
    application.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, echo))
    application.add_handler(MessageHandler(filters.PHOTO, handle_photo))

    scheduler_thread = threading.Thread(target=schedule_jobs)
    scheduler_thread.start()

    application.run_polling(allowed_updates=filters.Update.ALL_TYPES, drop_pending_updates=SHOULD_DROP_MESSAGES)


if __name__ == "__main__":
    main()
