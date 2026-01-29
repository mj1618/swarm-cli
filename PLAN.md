# Overview

Users will keep prompts in their projects under ./swarm/prompts/{name}.md
This is all the user will provide on the filesystem.

# CLI tool

The cli tool should where-applicable follow docker command syntax.

Each task gets a Task ID and it should be called "task" ID in all cases (to distinguish from OS process ID/PID).

The CLI allows the following functions:
1. Start a single agent (see swarm/bin/agent.sh for an example) - choosing a prompt from swarm/prompts/ - the user should be able to select a model (one of the models from running the command shown by examples/list-models.txt, default to opus-4.5-thinking)
2. Start a single agent that runs in a loop for a number of iterations (specified by the user, default 20). Again the user can select a model (default opus-4.5-thinking).

# Managing running agents

The user should be able to list all their running agents, and then select one to view it's logs, change the number of iterations for it, change the model, terminate the agent immeditaly or terminate it after it's current iteration.

All the state and status of the agent should be available when viewing the list.

# Pretty log line

`examples/pretty-log-line.sh` will need to be converted to golang in order to pretty-print the log lines coming out of the agent.
Note that an error in the pretty-log-line should never cause a termination of the agent, instead just the raw line should be displayed.

# Notes

A termination in the agent should never cause a termination of the loop, it should run another agent and continue on.
