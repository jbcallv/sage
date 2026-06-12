// orchestrator may greet the target
permit(
  principal == Agent::"orchestrator",
  action == Action::"GET",
  resource == Resource::"/hello"
);

// orchestrator may delegate a task to the worker (the delegation itself is legitimate)
permit(
  principal == Agent::"orchestrator",
  action == Action::"POST",
  resource == Resource::"/task"
);

// worker may list tasks
permit(
  principal == Agent::"worker",
  action == Action::"GET",
  resource == Resource::"/tasks/list"
);

// worker may never read admin data (the confused deputy target)
forbid(
  principal == Agent::"worker",
  action,
  resource == Resource::"/admin/secret-data"
);
