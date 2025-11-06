package Homework_4

//algorithm here

/*
 ---=== PSEUDO CODE ===---

On initialisation do
 state := RELEASED;
End on
On enter do
 state := WANTED;
 “multicast ‘req(T,p)’”, where T := LAMPORT time of ‘req’ at p
 wait for N-1 replies
 state := HELD;
End on
On receive ‘req (Ti,pi)’do
 if(state == HELD ||
 (state == WANTED &&
 (T,pme) < (Ti,pi)))
 then queue req
 else reply to req
End on
On exit do
 state := RELEASED
 reply to all in queue
End on
*/
