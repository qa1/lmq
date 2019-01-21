#!/usr/bin/python  -*- coding: utf-8 -*-

"""
	Lightweight Message Queue Library, version 1.1.0

	Copyright (C) 2018 Misam Saki, http://misam.ir
	Do not Change, Alter, or Remove this Licence
"""

import requests


class LightweightMessageQueueLibrary:

	def __init__(self, address='localhost', port=3000):
		self._url = 'http://%s:%s' % (address, port)

	@staticmethod
	def version():
		return '1.1.0'

	@staticmethod
	def compatible_version():
		return '1.2.0'

	@staticmethod
	def copyright():
		print('''
			***
			Lightweight Message Queue Library, version 1.1.0
			
			Copyright (C) 2018 Misam Saki, http://misam.ir
			Do not Change, Alter, or Remove this Licence
			***
		'''.replace('\t', ''))

	def help(self):
		req = requests.get('%s/help' % self._url)
		if req.status_code == 200:
			return req.text
		else:
			raise Exception(req.status_code, req.text)

	def list(self):
		req = requests.get('%s/list' % self._url)
		if req.status_code == 200:
			return req.text
		else:
			raise Exception(req.status_code, req.text)

	def count(self, queue):
		req = requests.get('%s/count/%s' % (self._url, queue))
		if req.status_code == 200:
			return int(req.text)
		else:
			raise Exception(req.status_code, req.text)

	def skip(self, queue, number):
		req = requests.get('%s/skip/%s/%s' % (self._url, queue, number))
		if req.status_code == 200:
			return 0
		else:
			raise Exception(req.status_code, req.text)

	def set(self, queue, message):
		req = requests.get('%s/set/%s/%s' % (self._url, queue, message))
		if req.status_code == 200:
			return 0
		else:
			raise Exception(req.status_code, req.text)

	def get(self, queue):
		req = requests.get('%s/get/%s' % (self._url, queue))
		if req.status_code == 200:
			return {
				'uid': req.headers.get('Uid', None),
				'message': req.text
			}
		else:
			raise Exception(req.status_code, req.text)

	def fetch(self, queue):
		req = requests.get('%s/fetch/%s' % (self._url, queue))
		if req.status_code == 200:
			return {
				'uid': req.headers.get('Uid', None),
				'message': req.headers.get('Message', None),
				'content': req.content
			}
		else:
			raise Exception(req.status_code, req.text)

	def download(self, message):
		req = requests.get('%s/download/%s' % (self._url, message))
		if req.status_code == 200:
			return req.content
		else:
			raise Exception(req.status_code, req.text)

	def delete(self, queue):
		req = requests.get('%s/delete/%s' % (self._url, queue))
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
