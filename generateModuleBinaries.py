#!/usr/bin/env python
import sys
import subprocess
import traceback
import glob
import os.path

def create_binary_name(source, os_name):
  binary_name = source.split("/")
  binary_name[2] = os_name
  return "/".join(binary_name)

recompile_all = False
if len(sys.argv) == 2:
  recompile_all = True

try:
  for go_file in glob.glob("modules/*/*.go"):
    for os_name in ["linux", "darwin"]:
      binary_name = create_binary_name(go_file, os_name)
      if recompile_all or not os.path.isfile(binary_name):
        cmd = "GOOS=%s go build -o %s %s" % (os_name, binary_name, go_file)
        print 'Executing => "%s"' % cmd
        p = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        stdout, stderr = p.communicate()
        if p.returncode > 0:
          raise Exception(stderr)
        print "Successfully created %s" % binary_name 
      else:
        print "'%s' already present.  Pass in 1 as the first parameter to recompile all go modules" % binary_name 
except Exception as e:
  if e:
    print "Error - %s" % e 
  else:
    traceback.print_exec
