# Lightweight Message Queue

A lightweight message queue to work with short messages or content references as message.

Version 1.4.6

## Usage
- Build server code
    > go build ./LMQ.go
- Build cleanup code
    > go build ./cleanup.go
- Run LMQ server and use by
    > ./LMQ ./config.json
- Run cleanup
    > ./cleanup ./config.json

- Use [PyLMQ](https://github.com/misamplus/PyLMQ) to connect

## Methods

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

## Message types

- [message]
    > Pure text message (for short messages).

- file:[file_path]
	> File content as the message.

- mysql:[table_name]/[id]
	> Mysql record as the message ('id' field as identification and 'data' field as content).
	
## License

Lightweight Message Queue

Copyright (C) 2018  Misam Saki, misam.ir

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
