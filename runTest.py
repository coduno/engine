#!/bin/python

import sys
import os

from subprocess import Popen, PIPE	
from threading import Thread

# Import Queue, module renamed in python 3.x
try:
	from Queue import Queue, Empty
except ImportError:
	from queue import Queue, Empty

# Read out line by line and put it in queue, close when finished
def read_output(out, queue):
	for line in iter(out.readline, b''):
		queue.put(line)
	out.close()

# Create a queue and begin reading out into queue items in a new thread
def start_read_queue(out):
	q = Queue()
	t = Thread(target=read_output, args=(out, q))
	t.daemon = True
	t.start()
	return q

# Log the output, stub for cloud datastore logic
def log_output(line, src):
	print src + line

# Check if queue out has a new item and push it to src and log it
# src is used to identify which output queue was used
def check_output(out, src, dest):
	try:
		line = out.get_nowait()
		log_output(line, src)
		if dest is not None:
			dest.write(line)
	except Empty:
		return False, None
	except IOError as e:
		return False, e
	return True, None

print os.getcwd()
print sys.argv[1]

	
#p1 = Popen(["docker", "run", "."], stdout=PIPE, stderr=PIPE, stdin=PIPE)
#out_queue = start_read_queue(p1.stdout)
#err_queue = start_read_queue(p1.stderr)
#p2 = Popen(["docker", "run", "."], stdin=PIPE, stdout=PIPE, stderr=PIPE)
#out2_queue = start_read_queue(p2.stdout)
#err2_queue = start_read_queue(p2.stderr)

#while ((not p1.stdout.closed) or (not out_queue.empty()) 
#	or (not err_queue.empty()) or (not p2.stdout.closed) or 
#	(not out2_queue.empty()) or (not err2_queue.empty())):
#
#	ret, err = check_output(out_queue, "OUT1", p2.stdin)
#	if not ret and err is not None:
#		break
#
#	ret, err = check_output(out2_queue, "OUT2", p1.stdin)
#	if not ret and err is not None:
#		break
#	
#	ret, err = check_output(err_queue, "ERR", None)
#	if ret or err is not None:
#		break
#	ret, err = check_output(err2_queue, "ERR", None)
#	if ret or err is not None:
#		break


