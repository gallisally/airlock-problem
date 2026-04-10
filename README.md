# The Airlock Problem
A classic problem to understand concurrency, parallelism, race conditions and deadlocks.  
This repository was created for my team, so everyone can explore the above mentioned problem space.  
You can use your favorite programming language.
I recommend forking this repository with your account, so it will list your solution in the fork-list and others can explore all the different solutions to this problem.

## The task
Let's say we are in the ISS and one of our astronauts is in EVA (floating outside in space) and wants to go back in. 
At the same time an astronaut wants to get out into space.  
We have only one airlock.  
If both doors open at the same time, the whole crew inside the ISS dies.  
If both doors keep being locked, the astronaut outside in space or/and in the airlock die.  
The airlock has three states/variables: 
- Outside door: bool (open/closed)
- Inside door: bool (open/closed)
- Chamber: enum (pressurized/empty/"changing state")

To avoid catastrophic failure, you need to follow these three simple rules:
- The chamber should only be able to (de-)pressurize when both doors are closed (and keep being closed).
- The outside door should only open when the chamber is empty.
- The inside door should only open when the chamber is pressurized.

Astronauts are threads/processes who run in parallel.
Both can access their respective door at the same time.
Remember, one Astronaut is outside, the other inside.

Implement the airlock system and the two astronauts.
You can use semaphores/mutexes/locks, whatever is available in your favorite programming language (I recommend mutex, if available).
Be aware that some languages (like Python) have global locks and can run only one thread at a time (single-threaded).
Please explore what library you can use to run astronauts and chamber in parallel.
For Python you can use the [multiprocessing library](https://docs.python.org/3/library/multiprocessing.html#module-multiprocessing).
