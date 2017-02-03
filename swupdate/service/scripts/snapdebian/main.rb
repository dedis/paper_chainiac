#!/usr/bin/env ruby
#
#
require 'optparse'
require 'optparse/time'

require_relative 'log'
require_relative 'scrapper'
require_relative 'processor'

$opts = {:packages => [],
         :since => Time.new(2005,03,12),
         :until => Time.now ,
         :folder => "snapshots"}

parser = OptionParser.new do |opts|
    opts.banner = "Debian snapshot scrapper"
    opts.on("-s","--since TIME","Scraps from given time ISO 8601 format (%Y-%m-%d)") do |t|
        $opts[:since] = Time.strptime(t,"%F")
    end

    opts.on("-u","--until TIME","Scraps until given time ISO 8601 format") do |t|
        $opts[:until] = Time.strptime(t,"%F")
    end
    opts.on("-v","--verbose","Verbosity to debug") do |v|
        $logger.level = Logger::DEBUG
    end

end

parser.parse!

# args read the arguments either from STDIN or the file and puts the list of
# packages into the :packages key in @@opts
def check_args
    if STDIN.tty? && ARGV.empty?
        $logger.info "No packages list given. Will retrieve all packages (time consuming)."
        return
    end
    reader = STDIN.tty? ? ARGV.shift : STDIN
    reader.each_line do |line|
        $opts[:packages] << line.strip
    end
end

def main
    check_args
    $logger.info "Crawling range #{$opts[:since]} to #{$opts[:until]}"
    scrapper =  Scrapper.new $opts[:packages], $opts[:since], $opts[:until]
    links = scrapper.scrap

    # ugly HACK to download also the release files but not process them
    links.each {|link|
        filen = File.join("snapshots/cache/"+extract_date(link[1][:release]) + "_" + extract_file(link[1][:release]))
        if !File.exists? filen
            while true do
                begin
                    File.open(filen,"w") do |f|
                        open(link[1][:release],"User-Agent" => "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/53.0.2785.92 Safari/537.36") do |l|
                            IO.copy_stream(l,f)
                        end
                    end
                    break
                rescue OpenURI::HTTPError
                    $logger.info "Error trying to download #{link[1][:release]}"
                    sleep 0.2
                    next
                end
            end
            $logger.debug "File #{filen} has been downloaded"
        else
            $logger.debug "File #{filen} is already cached"
        end
    }
    formatter = Formatter.new $opts[:folder],$opts[:packages]
    formatter.format links
end
## return the date compressed like YEARMONTHDAYHOURMINUTESECOND
def extract_date link 
    p=/([0-9]{4})([0-9]{2})([0-9]{2})T([0-9]{2})([0-9]{2})([0-9]{2})Z/
    res = link.to_s.match p
    raise "no date inside" unless res
    return res.to_a[1..-1].join""
end

def extract_file link
    p = /\/(\w+)$/
    res = link.to_s.match p
    raise "no file inside link" unless link.to_s.match p
    return $1
end

def help
    puts 'This ruby script will parse the website http://snapshot.debian.org/.
It takes a list of packages it needs to crawl, a start date and a end date.
The output is a csv file which is organized as:
snapshot_time, pkgName, version'
end

main
