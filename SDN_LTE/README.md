## SDN EPC User Guide

The manual contains all the steps to setup the SDN-based EPC. The entire setup (see Fig. 1) is primarily composed of the following components:

1. Floodlight controller
2. MySQL database (for HSS)
3. RAN simulator
4. Open vSwitch (for the gateways)
5. Sink program

<!--<img src="https://github.com/jain7aman/SDN_LTE/blob/master/SDN_LTE/images/sdn_epc_arch.png" alt="Fig. 1: SDN-based LTE EPC implementation" width="200" height="200" />-->

We need 6 machines in total to have the entire setup running.

## Requirements
* Ubuntu 14.04 (64-bit)
* RAM: 4 GB

## Controller Machine (MySQL and Floodlight)
1. Install MySQL database using the following command:
```
$ sudo apt-get install mysql-server
```
2. Download the dependencies for Floodlight: Floodlight master has been updated (on 04/30/16) to Java 8.
  * To download Java 8, run the following commands:
    ```
    
    $ sudo add-apt-repository ppa:webupd8team/java
    $ sudo apt-get update
    $ sudo apt-get install oracle-java8-installer
    ```
    To automatically set Java 8 environment variables, install the following package:
    ```
    $ sudo apt-get install oracle-java8-set-default
    ```
   



