#!/usr/bin/python3
# -*- coding: utf-8 -*-

import requests


class LightweightMessageQueueLibrary:

	def __init__(self, hosts='http://localhost:3000'):
		if type(hosts) is not list:
			hosts = [hosts]
		self._hosts = hosts
		self._round_robin_index = -1

	def help(self, host_index=0):
		req = requests.get('%s/help' % self._hosts[host_index])
		if req.status_code == 200:
			return req.text
		else:
			raise Exception(req.status_code, req.text)

	def list(self, host_index=0):
		req = requests.get('%s/list' % self._hosts[host_index])
		if req.status_code == 200:
			return req.text
		else:
			raise Exception(req.status_code, req.text)

	def count(self, queue, host_index=0):
		req = requests.get('%s/count/%s' % (self._hosts[host_index], queue))
		if req.status_code == 200:
			return int(req.text)
		else:
			raise Exception(req.status_code, req.text)

	def skip(self, queue, number, host_index=0):
		req = requests.get('%s/skip/%s/%s' % (self._hosts[host_index], queue, number))
		if req.status_code == 200:
			return 0
		else:
			raise Exception(req.status_code, req.text)

	def set(self, queue, message, host_index=0):
		req = requests.get('%s/set/%s/%s' % (self._hosts[host_index], queue, message))
		if req.status_code == 200:
			return 0
		else:
			raise Exception(req.status_code, req.text)

	def get(self, queue, host_index=None):
		if host_index is not None:
			req = requests.get('%s/get/%s' % (self._hosts[host_index], queue))
			if req.status_code == 200:
				return {
					'host_index': host_index,
					'uid': req.headers.get('Uid', None),
					'message': req.text
				}
			else:
				raise Exception(req.status_code, req.text)
		else:
			last_exception = Exception('Nothing!')
			indices = list(range(self._round_robin_index + 1, len(self._hosts))) + list(range(0, self._round_robin_index + 1))
			for index in indices:
				self._round_robin_index = index
				try:
					return self.get(queue, index)
				except Exception as exception:
					last_exception = exception
			raise last_exception

	def fetch(self, queue, host_index=None):
		if host_index is not None:
			req = requests.get('%s/fetch/%s' % (self._hosts[host_index], queue))
			if req.status_code == 200:
				return {
					'host_index': host_index,
					'uid': req.headers.get('Uid', None),
					'message': req.headers.get('Message', None),
					'content': req.content
				}
			else:
				raise Exception(req.status_code, req.text)
		else:
			last_exception = Exception('Nothing!')
			indices = list(range(self._round_robin_index + 1, len(self._hosts))) + list(range(0, self._round_robin_index + 1))
			for index in indices:
				self._round_robin_index = index
				try:
					output = self.fetch(queue, index)
					return output
				except Exception as exception:
					last_exception = exception
			raise last_exception

	def download(self, message, host_index=0):
		req = requests.get('%s/download/%s' % (self._hosts[host_index], message))
		if req.status_code == 200:
			return req.content
		else:
			raise Exception(req.status_code, req.text)

	def delete(self, queue, host_index=0):
		req = requests.get('%s/delete/%s' % (self._hosts[host_index], queue))
		if req.status_code == 200:
			return 0
		else:
			raise Exception(req.status_code, req.text)


if __name__ == "__main__":
	lmql = LightweightMessageQueueLibrary()
	try:
		res = lmql.help()
		print(res)
	except Exception as exception:
		print(exception)