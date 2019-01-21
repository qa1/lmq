#!/usr/bin/python  -*- coding: utf-8 -*-

"""
	Lightweight Message Queue Library, version 1.0.3

	Copyright (C) 2018 Misam Saki, http://misam.ir
	Do not Change, Alter, or Remove this Licence
"""

import requests


class LightweightMessageQueueLibrary:

	def __init__(self, address='localhost', port=3000):
		self._url = 'http://%s:%s' % (address, port)

	@staticmethod
	def version():
		return '1.0.3'

	@staticmethod
	def compatible_version():
		return '1.2.0'

	@staticmethod
	def copyright():
		print('''
			***
			Lightweight Message Queue Library, version 1.0.3
			
			Copyright (C) 2018 Misam Saki, http://misam.ir
			Do not Change, Alter, or Remove this Licence
			***
		'''.replace('\t', ''))

	def help(self):
		try:
			req = requests.get('%s/help' % self._url)
			if req.status_code == 200:
				return req.text
			else:
				return Exception(req.status_code, req.text)
		except Exception as exception:
			return exception

	def list(self):
		try:
			req = requests.get('%s/list' % self._url)
			if req.status_code == 200:
				return req.text
			else:
				return Exception(req.status_code, req.text)
		except Exception as exception:
			return exception

	def count(self, queue):
		try:
			req = requests.get('%s/count/%s' % (self._url, queue))
			if req.status_code == 200:
				return int(req.text)
			else:
				return Exception(req.status_code, req.text)
		except Exception as exception:
			return exception

	def skip(self, queue, number):
		try:
			req = requests.get('%s/skip/%s/%s' % (self._url, queue, number))
			if req.status_code == 200:
				return 0
			else:
				return Exception(req.status_code, req.text)
		except Exception as exception:
			return exception

	def set(self, queue, message):
		try:
			req = requests.get('%s/set/%s/%s' % (self._url, queue, message))
			if req.status_code == 200:
				return 0
			else:
				return Exception(req.status_code, req.text)
		except Exception as exception:
			return exception

	def get(self, queue):
		try:
			req = requests.get('%s/get/%s' % (self._url, queue))
			if req.status_code == 200:
				return {
					'uid': req.headers.get('Uid', None),
					'message': req.text
				}
			else:
				return Exception(req.status_code, req.text)
		except Exception as exception:
			return exception

	def fetch(self, queue):
		try:
			req = requests.get('%s/fetch/%s' % (self._url, queue))
			if req.status_code == 200:
				return {
					'uid': req.headers.get('Uid', None),
					'message': req.headers.get('Message', None),
					'content': req.content
				}
			else:
				return Exception(req.status_code, req.text)
		except Exception as exception:
			return exception

	def download(self, message):
		try:
			req = requests.get('%s/download/%s' % (self._url, message))
			if req.status_code == 200:
				return req.content
			else:
				return Exception(req.status_code, req.text)
		except Exception as exception:
			return exception

	def delete(self, queue):
		try:
			req = requests.get('%s/delete/%s' % (self._url, queue))
			if req.status_code == 200:
				return 0
			else:
				return Exception(req.status_code, req.text)
		except Exception as exception:
			return exception


if __name__ == "__main__":
	lmql = LightweightMessageQueueLibrary()
	res = lmql.help()
	if type(res) is not Exception:
		print(res)
	else:
		raise res
