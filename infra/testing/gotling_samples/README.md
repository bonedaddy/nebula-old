# gotling_samples

Contains sample files for [gotling](https://github.com/eriklupander/gotling)

# files

* `nebula_udp_load_1.yml`
  * Basic udp load test against the port a nebula node listens on
  * simulates 100 clients each sending 1000 requests with a variety of payload sizes
* `nebula_udp_load_2.yml`
  * Same test as `nebula_udp_load_1.yml` except significantly more intense
  * designed to put a lot of pressure on the nebula node garbage collector

  # usage

  Install gotling and run the following commands
  
  ```shell
  $> make run-1 # runs load_1.yml
  $> make run-2 # runs load_2.yml
  $> make run-all # runs load_1 and then load_2
  ```