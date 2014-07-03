#!/usr/bin/ruby
#

require 'socket'
require 'json'

$WINDOW_LENGTH = 10
$WINDOW_INTERVAL = 5
$DEFAULT_SCALE_UP_THRESHOLD = 2
$routes_file = "/var/lib/containers/router/routes.json"
$router_endpoint = "/var/lib/containers/router/broker.sock"
$stat_sock = nil
$router_sock = nil
$be_map = {}
$traffic_data = {}

def getstat()
	if 0==1
		if $stat_sock.nil?
			$stat_sock = UNIXSocket.new("/var/lib/haproxy/run/haproxy.sock") 
			$stat_sock.puts("prompt")
		end
		$stat_sock.puts("show stat -1 2 -1")
		$stat_sock.recv
	else
        	`echo "show stat -1 2 -1" | socat /var/lib/haproxy/run/haproxy.sock stdio | cut -d "," -f1,5,20`
	end
end

def load_data_set()
	data = JSON.parse(File.read($routes_file))
	data.each { |appname,approutes|
		$traffic_data[appname] = { "scount" => [], "slope" => [], "scale_up_threshold" => (approutes["scale_up_threshold"] || $DEFAULT_SCALE_UP_THRESHOLD), "scale_down_threshold" => 0 }
	}
	puts "Inserted #{$traffic_data.keys.inspect} for monitoring by autoscaler"
end

def main()
	$router_sock = UNIXSocket.new($router_endpoint)
        while true do 
                sleep($WINDOW_INTERVAL)
                #print "Check #{i}"
                data = getstat.split("\n")
                data.each { |backend|
                        be_name,scur,act = backend.split(",")
                        # lookup be
                        appname = be_name[3..-1] rescue nil
                        next if $traffic_data[appname].nil?
                        register_traffic(appname, Integer(Integer(scur)/Integer(act)), Integer(act))
                }
        end
end

def register_traffic(appname, scur, active_count)
	## Implements http://en.wikipedia.org/wiki/Theil-Sen_estimator
        tarray = $traffic_data[appname]["scount"]
        sarray = $traffic_data[appname]["slope"]
        tarray.push(scur)
	sarray.push(tarray[-1]-tarray[-2]) if tarray.length > 1
        if tarray.length > $WINDOW_LENGTH
		# TODO: insertion sort
		sarray.sort!
		median_slope = sarray[Integer($WINDOW_LENGTH/2)+1]
		puts "Scur : #{scur}, Median : #{median_slope}, tarray : #{tarray}, sarray : #{sarray}"
		prediction = scur+median_slope
		if prediction >= $traffic_data[appname]["scale_up_threshold"]
			# scale up
			# and now, as flap protection, clear the array
			puts "Scaling up #{appname}"
			$router_sock.puts("scale-up #{appname}")
			$traffic_data[appname]["scount"] = []
			$traffic_data[appname]["slope"] = []
		elsif prediction <= $traffic_data[appname]["scale_down_threshold"] && active_count > 1
			# scale down
			puts "Scaling down #{appname}"
			$router_sock.puts("scale-down #{appname}")
			# and now, as flap protection, clear the array
			$traffic_data[appname]["scount"] = []
			$traffic_data[appname]["slope"] = []
		end
		tarray.pop()
		sarray.pop()
		puts "Prediction for #{appname} : #{prediction}"
        end
end 

load_data_set()
main()
