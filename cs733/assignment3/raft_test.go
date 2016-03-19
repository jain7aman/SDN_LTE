package main

import (
	"log"
	"os"
	"strconv"
	"testing"
	"time"
)

/*
* In this test case we test the scenario in which leader machine shuts down, then new leader is elected from remaining servers.
* After some time second leader also fails, leaving the number of servers to be 3. Now also, new leader should get elected, which is
* eventually shut down, leaving the number of servers to be just 2. Now no new leader should get elected.
 */
func Test_MultipleLeaderFailures(t *testing.T) {
	// delete previous logs, if any
	deleteLogs(5)
	rafts := makeRafts("Cluster_config3.json") // array of []raft.Node

	ldr := getLeader(rafts)
	leader1 := ldr.Id()
	ldr.Append([]byte("Msg_ID:1@@Command1"))

	// First Leader Failure, new leader should get elected
	ldr.Shutdown()

	ldr = getLeader(rafts)
	leader2 := ldr.Id()

	if leader1 == leader2 {
		log.Fatal("No new leader elected")
	}

	// Second Leader Failure, new leader should get elected
	ldr.Shutdown()

	ldr = getLeader(rafts)
	leader3 := ldr.Id()

	if leader3 == leader2 || leader3 == leader1 {
		log.Fatal("No new leader elected")
	}
	//tool cover -html=coverage.out
	//test -v -gcflags "-N -l" ss  -coverprofile=coverage.out
	// Third Leader Failure, no new leader should get elected
	ldr.Shutdown()
	// waits for leader to get elected
	time.Sleep(3 * time.Second)

	ldr = checkLeader(rafts)

	if ldr.Id() != -1 {
		log.Fatal("No new leader should have been elected, leader1 = %v leader2 = %v leader3 = %v leader4 = %v", leader1, leader2, leader3, ldr.Id())
	}

	for i := 0; i < len(rafts); i++ {
		if i+1 != leader1 && i+1 != leader2 && i+1 != leader3 {
			rafts[i].(*RaftServer).Shutdown()
		}
		rafts[i].(*RaftServer).ClearLogs()
	}
}

/*
* This test case tests the replication of a single message in a failure-free case.
* It inserts a message using leader.Append, and expects to see it show up all raft
* nodes’ commit channels.
* test -c -v -gcflags "-N -l" ./...
 */
func Test_Append(t *testing.T) {
	// delete previous logs, if any
	deleteLogs(5)
	rafts := makeRafts("Cluster_config1.json") // array of []raft.Node
	ldr := getLeader(rafts)
	ldr.Append([]byte("Msg_ID:1@@Command1"))
	time.Sleep(1 * time.Second)

	checkReplication(t, rafts, "Msg_ID:1@@Command1", "Test_Append")

	for i := 0; i < len(rafts); i++ {
		rafts[i].(*RaftServer).Shutdown()
		rafts[i].(*RaftServer).ClearLogs()
	}
	//time.Sleep(3 * time.Second)
}

/*
* This method checks the response of Candidate to Append entry request by a client.
* We should receive Error response on COmmit Channel
 */
func Test_CandidateAppendEntries(t *testing.T) {
	// delete previous logs, if any
	deleteLogs(5)
	rafts := makeRafts("Cluster_config2.json") // array of []raft.Node

	cand := getACandidate(rafts)
	cand.Append([]byte("Msg_ID:1@@Command1"))
	time.Sleep(1 * time.Second)

	select {
	case ci := <-cand.CommitChannel():
		commitAction := ci.(Commit)
		if commitAction.Err == nil {
			log.Fatal("Client Should have received an error")
		}
	default:
		log.Fatal("ERROR:: Expected error message on client channel")
	}

	for i := 0; i < len(rafts); i++ {
		rafts[i].(*RaftServer).Shutdown()
		rafts[i].(*RaftServer).ClearLogs()
	}

}

/*
* Here we shutdown a leader so new leader gets elected and then we wake up the old leader
* after some time state of old leader should not remain LEADER
 */
func Test_LeaderReincarnation(t *testing.T) {
	// delete previous logs, if any
	deleteLogs(5)
	rafts := makeRafts("Cluster_config3.json") // array of []raft.Node

	ldr := getLeader(rafts)
	leader1 := ldr.Id()
	ldr.Append([]byte("Msg_ID:1@@Command1"))
	time.Sleep(1 * time.Second)

	// making sure the entry is replicated to all servers
	checkReplication(t, rafts, "Msg_ID:1@@Command1", "Test_LeaderReincarnation")

	// First Leader Failure, new leader should get elected
	// we are not calling shutdwon method because we want don't want to close its socket connections
	ldr.Shutdown()
	//	ldr.ReceiveChannel <- ShutDownEvent{Id: ldr.ID}
	time.Sleep(2 * time.Second)

	newldr := getLeader(rafts)
	leader2 := newldr.Id()

	if leader1 == leader2 {
		log.Fatal("No new leader elected")
	}

	// lets bring old leader back to life
	ldr = reviveServer(ldr.Id(), "Cluster_config3.json")
	ldr.State = LEADER // change the state from POWEROFF to LEADER
	go ldr.NodeStart()
	time.Sleep(3 * time.Second)

	// now two leaders in the system
	// old leader should step down to follower
	numLeaders := getNumberOfLeaders(rafts)

	if numLeaders > 1 {
		log.Fatal("Old leader still active, should have abdicated the throne, numLeaders = ", numLeaders)
	}

	for i := 0; i < len(rafts); i++ {
		if i+1 != ldr.Id() {
			rafts[i].(*RaftServer).Shutdown()
			rafts[i].(*RaftServer).ClearLogs()
		}
	}
	ldr.Shutdown()
	ldr.ClearLogs()
}

/*
* This method checks the response of follower ( who previously was a leader ) to Append entry request by a client.
* We should receive Error response on COmmit Channel
 */
func Test_FollowerAppendEntries(t *testing.T) {
	// delete previous logs, if any
	deleteLogs(5)
	rafts := makeRafts("Cluster_config4.json") // array of []raft.Node

	oldLdr := getLeader(rafts)
	leader1 := oldLdr.Id()
	oldLdr.Append([]byte("Msg_ID:1@@Command1"))
	time.Sleep(1 * time.Second)

	// making sure the entry is replicated to all servers
	checkReplication(t, rafts, "Msg_ID:1@@Command1", "Test_LeaderReincarnation")

	// First Leader Failure, new leader should get elected
	// we are not calling shutdwon method because we want don't want to close its socket connections
	oldLdr.Shutdown()
	//	oldLdr.ReceiveChannel <- ShutDownEvent{Id: oldLdr.ID}
	time.Sleep(2 * time.Second)

	newldr := getLeader(rafts)
	leader2 := newldr.Id()

	if leader1 == leader2 {
		log.Fatal("No new leader elected")
	}

	// lets bring old leader back to life
	oldLdr = reviveServer(oldLdr.Id(), "Cluster_config4.json")
	oldLdr.State = LEADER // change the state from POWEROFF to LEADER
	go oldLdr.NodeStart()
	time.Sleep(3 * time.Second)

	// now two leaders in the system
	// old leader should step down to follower
	if oldLdr.State == LEADER {
		log.Fatal("Old leader still active, should have abdicated the throne")
	}

	oldLdr.Append([]byte("Msg_ID:2@@Command2"))
	time.Sleep(1 * time.Second)

	select {
	case ci := <-oldLdr.CommitChannel():
		commitAction := ci.(Commit)
		if commitAction.Err == nil {
			log.Fatal("Client Should have received an error")
		}
	default:
		log.Fatal("ERROR:: Expected error message on client channel")
	}
	for i := 0; i < len(rafts); i++ {
		if i+1 != oldLdr.Id() {
			rafts[i].(*RaftServer).Shutdown()
			rafts[i].(*RaftServer).ClearLogs()
		}
	}
	oldLdr.Shutdown()
	oldLdr.ClearLogs()
}

/*
* This is the basic test case for partition. We partition the configuration of 5 servers into 3 and 2.
* Leader of unpartitioned configuration is in minority parition1 (size = 2). So new leader will be elected in majority partition1 (size = 3)
* We HEAL the parition. Now two leaders in the system, at last there should be only one leader that too of partition1,
* because its term would be higher
 */
func Test_Partition1(t *testing.T) {
	// delete previous logs, if any
	deleteLogs(5)
	cl, rafts := makeMockRafts("Cluster_config3.json") // array of []raft.Node
	ldr := getLeader(rafts)
	folw := getAFollower(rafts, ldr.Id())
	ldr.Append([]byte("Msg_ID:1@@Command1"))
	time.Sleep(1 * time.Second)

	checkReplication(t, rafts, "Msg_ID:1@@Command1", "Test_Partition")

	partition1 := make([]int, 3)
	newRafts := make([]Node, 3)
	cnt := 0
	for i := 0; i < len(rafts); i++ {
		if i+1 != ldr.Id() && i+1 != folw.Id() && cnt < 3 {
			partition1[cnt] = i + 1
			newRafts[cnt] = rafts[i]
			cnt++
		}
	}

	//leader is in the minority
	cl.Partition(partition1, []int{ldr.Id(), folw.Id()}) // Cluster partitions into two.
	time.Sleep(1 * time.Second)

	// new leader should have got elected in partition1
	newLdr := getLeader(newRafts)

	if newLdr.Id() == ldr.Id() {
		log.Fatal("No new leader got elected")
	}

	//heal the partition
	cl.Heal()
	// now two leaders in the system, at last there should be only one leader that too of partition1, because its term would be higher
	time.Sleep(200 * time.Millisecond)

	curLdr := getLeader(rafts)
	numLeaders := getNumberOfLeaders(rafts)

	if numLeaders != 1 {
		log.Fatal("More than 1 leader in the system")
	}

	if curLdr.Id() != newLdr.Id() {
		log.Fatal("Wrong leader got elected, leader id = ", curLdr.Id())
	}

	for i := 0; i < len(rafts); i++ {
		rafts[i].(*RaftServer).Shutdown()
		rafts[i].(*RaftServer).ClearLogs()
	}
}

/*
* IN this test case, we parition the system into two partitions of size 3 and 2 respectively. Leader is in majority parition1 (size 3).
* After the partition, there should not be any leader for minority parition 2 (size = 2). During this time we append various entries in partition1
* and get those committed. After which we HEAL the partiton.
* Now there will be RE ELECTION as partition2 severs would have higher term but their log is not updated and also they are in minority.
* So check that the new leader should not be any one of the minority partition2 servers (as their logs are not updated)
* After new leader ( any server from majority partition1) got elected, then the followers of minority partiton should cach up with the log of leader
* and after some time their log should be identical with leader's log
 */
func Test_Partition2(t *testing.T) {
	// delete previous logs, if any
	deleteLogs(5)
	cl, rafts := makeMockRafts("Cluster_config1.json") // array of []raft.Node

	ldr := getLeader(rafts)
	ldr.Append([]byte("Msg_ID:1@@Command1"))
	time.Sleep(1 * time.Second)

	checkReplication(t, rafts, "Msg_ID:1@@Command1", "Test_Partition")

	partition1 := make([]int, len(rafts)-2)
	newRafts := make([]Node, len(rafts)-2)
	cnt := 0

	//partition again, this time leader is in majority
	partition2 := make([]int, 2)
	newRafts2 := make([]Node, 2)
	partition1[0] = ldr.Id()
	newRafts[0] = rafts[ldr.Id()-1]
	cnt = 1
	cnt2 := 0
	for i := 0; i < len(rafts); i++ {
		if i+1 != ldr.Id() {
			if cnt < 3 {
				partition1[cnt] = i + 1
				newRafts[cnt] = rafts[i]
				cnt++
			} else {
				partition2[cnt2] = i + 1
				newRafts2[cnt2] = rafts[i]
				cnt2++
			}
		}
	}

	//	log.Printf("Partition1 size = %v Partition2 size = %v  newRafts2 size = %v \n", len(partition1), len(partition2), len(newRafts2))
	//	log.Printf(" Partition 1:->  (1)->%v (2)->%v (3)->%v   Partition 2:->  (1)->%v (2)->%v \n", partition1[0], partition1[1], partition1[2], partition2[0], partition2[1])

	//leader is in the minority
	cl.Partition(partition1, partition2) // Cluster partitions into two.
	//	Debug = true
	time.Sleep(1 * time.Second)

	//no new leader should get elected in minority
	numLeaders := getNumberOfLeaders(newRafts2)

	if numLeaders != 0 {
		log.Printf("No new leader should have got elected in minority partition, numleaders = %v leader id = %v", numLeaders, getLeader(newRafts2).Id())
		log.Fatal()

	}

	// append some entries in bigger partition
	ldr.Append([]byte("Msg_ID:2@@Command2"))
	time.Sleep(2 * time.Second)
	checkReplication(t, newRafts, "Msg_ID:2@@Command2", "Test_Partition2")

	ldr.Append([]byte("Msg_ID:3@@Command3"))
	time.Sleep(2 * time.Second)
	checkReplication(t, newRafts, "Msg_ID:3@@Command3", "Test_Partition3")

	ldr.Append([]byte("Msg_ID:4@@Command4"))
	time.Sleep(2 * time.Second)
	checkReplication(t, newRafts, "Msg_ID:4@@Command4", "Test_Partition4")

	ldr.Append([]byte("Msg_ID:5@@Command5"))
	time.Sleep(2 * time.Second)
	checkReplication(t, newRafts, "Msg_ID:5@@Command5", "Test_Partition5")

	time.Sleep(200 * time.Millisecond)
	//heal the partition
	cl.Heal()
	// now new election should take place and none of the followers in minority partition should be lected as leader
	time.Sleep(200 * time.Millisecond)

	newldr := getLeader(rafts)

	//	log.Printf(" New leader ID =%v \n", newldr.Id())

	for i := 0; i < len(partition2); i++ {
		if newldr.Id() == partition2[i] {
			log.Fatal("None of the followers/candidates of minority partition should have get elected as their log is not uptodate, new leader id= ", newldr.Id())
		}
	}
	time.Sleep(2 * time.Second)

	// followers/candidates in minority partition should catch up in log
	for i := 0; i < len(partition2); i++ {
		folw := rafts[partition2[i]-1].(*RaftServer)
		if len(folw.Log) == 5 {
			for j := 1; j <= 5; j++ {
				msg := string(folw.Log[j-1].Command)
				switch j {
				case 1:
					if msg != "Msg_ID:1@@Command1" {
						log.Fatal("expected Msg_ID:1@@Command1 found = ", msg)
					}
				case 2:
					if msg != "Msg_ID:2@@Command2" {
						log.Fatal("expected Msg_ID:2@@Command2 found = ", msg)
					}
				case 3:
					if msg != "Msg_ID:3@@Command3" {
						log.Fatal("expected Msg_ID:3@@Command3 found = ", msg)
					}
				case 4:
					if msg != "Msg_ID:4@@Command4" {
						log.Fatal("expected Msg_ID:4@@Command4 found = ", msg)
					}
				case 5:
					if msg != "Msg_ID:5@@Command5" {
						log.Fatal("expected Msg_ID:5@@Command5 found = ", msg)
					}
				}
			}
		} else {
			log.Fatal("Follower log should be up to date by now, length of folw log \n", len(folw.Log), " Follower id = ", partition2[i])
		}
	}

	for i := 0; i < len(rafts); i++ {
		rafts[i].(*RaftServer).Shutdown()
		rafts[i].(*RaftServer).ClearLogs()
	}
}

func checkReplication(t *testing.T, rafts []Node, expectedMsg string, testId string) {
	for i := 0; i < len(rafts); i++ {
		select {
		case ci := <-rafts[i].(*RaftServer).CommitChannel():
			commitAction := ci.(Commit)
			if commitAction.Err != nil {
				t.Fatal(commitAction.Err)
			}
			if string(commitAction.Data) != expectedMsg {
				log.Printf("ERROR:: Test ID = %v Got data %v expected = %v\n", testId, string(commitAction.Data), expectedMsg)
				log.Fatal()
			}
		default:
			log.Printf("ERROR:: Test ID = %v Expected message on all nodes, not found on server = %v\n", testId, rafts[i].(*RaftServer).Id())
			log.Fatal()
		}
	}
}

func deleteLogs(numServers int) {
	for i := 1; i <= numServers; i++ {
		if _, err := os.Stat("server_" + strconv.Itoa(i) + "_log"); err == nil {
			err = os.RemoveAll("server_" + strconv.Itoa(i) + "_log")
			if err != nil {
				panic(err)
			}
		}
		if _, err := os.Stat(strconv.Itoa(i) + "_state"); err == nil {
			err = os.RemoveAll(strconv.Itoa(i) + "_state")
			if err != nil {
				panic(err)
			}
		}
	}
}
