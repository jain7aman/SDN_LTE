## SDN EPC User Guide

The manual contains all the steps to setup the SDN-based EPC. The entire setup (see Fig. 1) is primarily composed of the following components:

1. Floodlight controller
2. MySQL database (for HSS)
3. RAN simulator
4. Open vSwitch (for the gateways)
5. Sink program

<!--<img src="https://github.com/jain7aman/SDN_LTE/blob/master/SDN_LTE/images/sdn_epc_arch.png" alt="Fig. 1: SDN-based LTE EPC implementation" width="200" height="200" />-->

We need 6 machines in total to have the entire setup running.

#### Requirements ####
* Ubuntu 14.04 (64-bit)
* RAM: 4 GB

#### Controller Machine (MySQL and Floodlight) ####
1. Install MySQL database using the following command:<br/>
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
    
    To automatically set Java 8 environment variables, install the following package:<br/>
    ```
    $ sudo apt-get install oracle-java8-set-default
    ```
  * To download remaining dependencies for Floodlight master and above:<br/>
    ```
    $ sudo apt-get install build-essential ant maven python-dev eclipse
    ```<br/>
3. To download dependencies for Floodlight v1.2 and below:<br/>
    ```
    $ sudo apt-get install build-essential openjdk-7-jdk ant maven python-dev eclipse
    ```
4. Download and build Floodlight: The below uses the master version of Floodlight. To use a specific version, specify the branch in the git step by appending -b <branch-name>.<br/>
    ```
    $ git clone https://github.com/floodlight/floodlight.git
    $ cd floodlight
    $ git submodule init
    $ git submodule update
    $ ant
     
    $ sudo mkdir /var/lib/floodlight
    $ sudo chmod 777 /var/lib/floodlight
    ```
5. Setting up Floodlight on Eclipse: The following command creates several files: Floodlight.launch, Floodlight_junit.launch, .classpath, and .project. From these you can setup a new Eclipse project.<br/>
    ```
    ant eclipse
    ```
    * Open eclipse and create a new workspace
    * **File -> Import -> General -> Existing Projects into Workspace**. Then click **Next**.
    * From **Select root directory** click **Browse**. Select the parent directory where you placed floodlight earlier.
    * Check the box for **Floodlight**. No other Projects should be present and none should be selected.
    * Click **Finish**.
6. Running Floodlight in Eclipse: Create the FloodlightLaunch target.
    * Click **Run->Run Configurations**
    * Right Click **Java Application->New**
    * For **Name** use **FloodlightLaunch**
    * For **Project** use **Floodlight**
    * For **Main** use ```net.floodlightcontroller.core.Main```
    * Click **Apply**
7. Creating modules for SDN EPC:
    * Expand the **floodlight** item in the Package Explorer and find the ```src/main/java``` folder.
    * Right-click on the ```src/main/java``` folder and choose **New/Package**.
    * Enter ```net.floodlightcontroller.sdnepc``` in the **Name** box and click on the Finish button. This creates a new package.
    * Copy the files Constants.java, MME.java, HSS.java, SGW.java, PGW.java and Utils.java and paste into the new package created.


   



