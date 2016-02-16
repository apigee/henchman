#!/usr/bin/env python
import sys
import subprocess
import traceback
import glob
import os.path

def create_binary_name(source, os_name):
  binary_name = source.split("/")
  binary_name[2] = binary_name[1]+"."+os_name
  return "/".join(binary_name)

def create_binary(os_name, binary_name, go_file):
  cmd = "GOOS=%s godep go build -o %s %s" % (os_name, binary_name, go_file)
  print 'Executing => "%s"' % cmd
  p = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
  stdout, stderr = p.communicate()
  if p.returncode > 0:
    raise Exception(stderr)
  print "Successfully created %s" % binary_name 

# pass 1 to recompile all
# pass 2 <module_name> to recompile one module
try:
  recompile = False
  compile_file = ""
  if len(sys.argv) > 1:
    if sys.argv[1] == "2":
      compile_file = sys.argv[2]
    recompile = True

  for go_file in glob.glob("modules/*/*.go"):
    if compile_file == "" or compile_file == go_file.split("/")[-1]:
      for os_name in ["linux", "darwin"]:
        binary_name = create_binary_name(go_file, os_name)
        if recompile or not os.path.isfile(binary_name):
          create_binary(os_name, binary_name, go_file)
        else:
          print "'%s' already present.  Pass in 1 as the first parameter to recompile all go modules" % binary_name 
except Exception as e:
  if e:
    print "Error - %s" % e 
  else:
    traceback.print_exec
