#!/usr/bin/env ruby

# Copyright 2015 The Zebu Authors. All rights reserved.

require 'optparse'
require 'ostruct'
require 'pp'
require 'mkmf'
require 'ptools'

$commands = {"compile" => "compile", "error" => "error", "run" => "run"}
$zebu = nil

def get_command(line) 
  toks = line.split
  unless (toks[0] == "//" && !$commands[toks[1]].nil?) 
    puts "error: unexpected command %s" % toks[1]
    exit 1
  end 
  return toks[1]
end

def compile_file(name) 
  output = `#{$zebu} #{name}`
  return output, $?.exitstatus
end

# compile command is simple: must return no output
# and no error.
def do_compile_command(name, file)
  output, result = compile_file(name)
  if (output != "" || result != 0)
    puts "----------------------------------------------------------------------"
    puts "BUG: %s failed to compile" % name
    puts "----------------------------------------------------------------------"
    puts output
    puts "----------------------------------------------------------------------"
    exit 1
  end
end

def parse_output(output)
  errors = {}
  output.lines do |line|
		ln = line.split(":")[1].to_i
		if errors[ln].nil?
			errors[ln] = [line.strip]
		else
			errors[ln] << line.strip
		end
  end
  return errors
end

# error command is much harder
def do_error_command(name, file) 
  output, result = compile_file(name)
  errors = parse_output(output)
  saved = []
  ln = 1  # first line is already parsed, start loop at 2
  file.each do |line| 
    ln = ln + 1
    match = line.match(/(\/\/+).*ERROR(.*)/)
    unless !match.nil? && match[1].length == 2 
			unless errors[ln].nil?
				errors[ln].each do |err|
					saved << [err, ""]
				end
			end
      next
    end

    regex = "#{name}:#{ln}:.+:.*#{match[2].strip}"
    regelse = "#{name}:#{ln}:.+:.*"

		if errors[ln].nil? 
			saved << ["", regex]
			next
		end

    errors[ln].each do |err|
      omatch = output.match(regex)
      unless omatch.nil? 
        next
			end
			saved << [err, regex]
    end
	end

  if saved.length > 0 
    puts "----------------------------------------------------------------------"
    puts "BUG: %s unmatched errors " % name
    puts "----------------------------------------------------------------------"
    saved.each do |error|
      puts "#{error[0]} -- #{error[1]}"
    end
    puts "----------------------------------------------------------------------"
    exit 1
  end
end

def do_run_command(name, file)
  puts "error: run not yet implemented"
  exit 1
end

options = OpenStruct.new
options.encoding = "utf8"
options.verbose = false

optparse = OptionParser.new do |opts| 
  opts.banner = "Usage: check.rb  -- [test] "

  opts.on("-v", "--verbose", "Run verbosely") do |v|
    options.verbose = v
  end

  opts.on("-h", "--help", "Show this message") do 
    puts opts
    exit
  end
end

optparse.parse!

if ARGV.size < 1 
  puts optparse.help
  exit
end

# Test for zebu in path
$zebu = File.which("zebu")
if $zebu.nil?
  puts "error: zebu not found"
  exit 1
end

name = ARGV[0]

if !File.file?(name)
  puts "file not found"
  exit 1
end

file = File.open(name)

command = get_command(file.first)

case command
when "compile"
  do_compile_command(name, file)
when "error"
  do_error_command(name, file)
when "run"
  do_run_command(name, file)
else
  puts "error: internal script error"
  exit 1
end

exit 0



