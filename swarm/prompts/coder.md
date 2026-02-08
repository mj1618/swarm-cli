# Context from planner

{{output:planner}}

# Find unfinished tasks

Use your agent id to find tasks that you left unfinished by looking for files named:
./swarm/todo/*.{your-swarm-agent-id}.processing.md

If you found a task to continue, skip the "Find a task to complete" section and jump straight to "Execute the task".

# Find a task to complete

Look under the ./swarm/todo/ folder for a file suffixed with ".pending.md" - and claim it as a task by renaming it to ".{your-swarm-agent-id}.processing.md".
Make sure the file you choose doesn't have any dependencies that are not completed.
Make sure your work won't conflict with other work that is already in progress.
If no tasks are found - skip the "Execute the task" step and go to the "If nothing to do" step.

# Implement the task

Then read the file and execute the task to completion, adding tests and testing when you think you need to.
When done, rename the file to ".completed.md" and add a note about what you did.
If you can\'t complete the task, rename the file to ".pending.md" and add a note about what went wrong to the file.
When you\'re done, `git add .`, `git commit -m "a suitable message` and `git push` your changes to the remote repository.

# If nothing to do

If there are no pending tasks found, your job is to test the feature and report any issues.
Then write those issues as new pending tasks in the swarm/todo/{feature-name}/ folder.
When you have done that exit immediately!
