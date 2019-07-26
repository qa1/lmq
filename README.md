# Lightweight Message Queue

A lightweight message queue to work with short messages or content references as message.

Version 1.4.2

# Methods

- GET /list
    > List of the queues.

- GET /count/[queue]
	> Number of messages in a queue.

- GET /skip/[queue]/[number]
	> Skip messages in the queue.

- GET /set/[queue]/[message]
	> Set a message in the queue.

- GET /get/[queue]
	> Get a message in the queue.

- GET /fetch/[queue]
	> Fetch a message with content in a queue.

- GET /download/[message]
	> Download content of the message.

- GET /delete/[queue]
	> Delete the queue.

# Message types

- [message]
    > Pure text message (for short messages).

- file:[file_path]
	> File content as the message.

- mysql:[table_name]/[id]
	> Mysql record as the message ('id' field as identification and 'data' field as content).
	
# License

Copyright (C) 2018 Misam Saki, http://misam.ir

Do not Change, Alter, or Remove this Licence
