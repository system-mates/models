(() => {
  const root = document.querySelector("[data-arch-view]");
  if (!root) return;

  const read = name => {
    const element = root.querySelector(`[data-${name}]`);
    if (!element) return null;
    try {
      const value = JSON.parse(element.textContent);
      if (typeof value !== "string") return value;
      try { return JSON.parse(value); }
      catch { return JSON.parse(atob(value)); }
    } catch { return null; }
  };
  const structure = read("structure") || {};
  const selectedModel = root.dataset.selectedModel || "";
  const mode = root.dataset.archView;
  const status = read("status");
  const events = read("events") || [];
  const nodes = [];
  const edges = [];
  const details = new Map();
  const seen = new Set();
  const producerIDs = new Map();
  const modelNames = new Map((structure.models || []).map(model => [model.id,model.name]));
  const selectedModelIDs = new Set([selectedModel]);
  const dependencyInfo = new Map();
  for (const coordinator of Object.values(structure.coordinators?.dependencies || {})) {
    for (const dependency of coordinator.dependencies || []) {
      const model = modelNames.get(coordinator.model) || coordinator.model || selectedModel;
      const id = `${dependency.kind}:${model}:${dependency.key}`;
      const entries = dependencyInfo.get(id) || [];
      entries.push({coordinator:coordinator.name, model, producers:dependency.producers || [], definition:coordinator});
      dependencyInfo.set(id, entries);
    }
  }
  for (const model of structure.models || []) {
    if (model.name === selectedModel || model.id === selectedModel) {
      selectedModelIDs.add(model.id);
      selectedModelIDs.add(model.name);
    }
  }

  const icons = {
    route: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#36566f" stroke-width="1.8" stroke-linecap="round"><rect x="4" y="4" width="16" height="6" rx="1.5"/><rect x="4" y="14" width="16" height="6" rx="1.5"/><path d="M8 7h.01M8 17h.01M12 7h5M12 17h5"/></svg>',
    schedule: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#36566f" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="5" width="18" height="16" rx="3"/><path d="M7 3v4m10-4v4M3 10h18"/><path d="M12 14v3l2 1"/></svg>',
    bootstrap: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#36566f" stroke-width="2" stroke-linecap="round"><path d="M12 2v9M6.3 5.7a8 8 0 1 0 11.4 0"/></svg>',
    producer: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M7 4.5v15l12-7.5z" fill="#334155"/></svg>',
    io: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M7 4.5v15l12-7.5z" fill="#c2410c"/></svg>',
    state: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#334155" stroke-width="1.6"><rect x="3" y="3" width="11" height="18" rx="2"/><path d="M14 3h7v9h-7M14 12h7v9h-7"/><circle cx="8" cy="9" r="1.7" fill="#334155"/><circle cx="9.5" cy="15" r="1.7" fill="#334155"/></svg>',
    share: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#334155" stroke-width="1.6" stroke-dasharray="1 2" stroke-linecap="round"><rect x="3" y="3" width="11" height="18" rx="2"/><path d="M14 3h7v9h-7M14 12h7v9h-7"/><circle cx="8" cy="9" r="1.7"/><circle cx="9.5" cy="15" r="1.7"/></svg>',
    model: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#475569" stroke-width="1.8"><path d="m12 2 9 5-9 5-9-5z"/><path d="m3 12 9 5 9-5M3 17l9 5 9-5"/></svg>',
    activity: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#64748b" stroke-width="1.8"><path d="M5 3h14v18H5zM8 8h8M8 12h8M8 16h5"/></svg>',
    html_activity: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#475569" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m8 7-5 5 5 5m8-10 5 5-5 5M14 4l-4 16"/></svg>'
  };
  const iconURI = type => `data:image/svg+xml,${encodeURIComponent(icons[type] || icons.activity)}`;
  const add = (id, type, label, value, extra = {}) => {
    if (!id || label == null || String(label).trim() === "" || seen.has(id)) return;
    seen.add(id);
    nodes.push({id,type,label:String(label),...extra});
    details.set(id,value);
  };
  const inModel = value => !selectedModel || selectedModelIDs.has(value?.model);
  const modelName = value => modelNames.get(value) || value || selectedModel || "application";
  const walkActivities = (activities, visit, path = []) => {
    for (const [index,activity] of (activities || []).entries()) {
      const activityPath = [...path, `${activity.name || activity.type || "activity"}-${index}`];
      visit(activity,activityPath);
      walkActivities(activity.activities,visit,[...activityPath,"iterate"]);
      walkActivities(activity.then_activities,visit,[...activityPath,"then"]);
      walkActivities(activity.else_activities,visit,[...activityPath,"else"]);
    }
  };

  for (const [name,producer] of Object.entries(structure.producers || {})) {
    if (!inModel(producer)) continue;
    const model = producer.model || "application";
    const id = `producer:${model}:${name}`;
    let io = false;
    walkActivities(producer.activities,activity => {
      if (activity.type === "io") io = true;
    });
    producerIDs.set(name,id);
    add(id,"producer",name,producer,{io,profiles:producer.profiles || []});
    walkActivities(producer.activities,(activity,activityPath) => {
        let targetID = id;
        if (activity.type === "html") {
          targetID = `activity:${model}:${name}:${activityPath.join(":")}`;
          add(targetID,"html_activity",activity.name,{...activity,model,producer:name});
          edges.push({from:id,to:targetID,type:"contains"});
        }
        for (const access of activity.accesses || []) {
          if (access.kind !== "state" && access.kind !== "share") continue;
          const resourceID = `${access.kind}:${model}:${access.key}`;
          add(resourceID,access.kind,access.key,{...access,model,producer:name,activity:activity.name});
          if (access.direction === "input" || access.direction === "both") {
            edges.push({from:resourceID,to:targetID,type:"read"});
          }
          if (access.direction === "output" || access.direction === "both") {
            edges.push({from:targetID,to:resourceID,type:"write"});
          }
        }
    });
  }

  for (const [name,producer] of Object.entries(structure.producers || {})) {
    if (!inModel(producer)) continue;
    const source = producerIDs.get(name);
    if (!source) continue;
    walkActivities(producer.activities,activity => {
      if (activity.type !== "call") return;
      for (const calledProducer of activity.run || []) {
        const target = producerIDs.get(calledProducer);
        if (target) edges.push({from:source,to:target,type:"invokes"});
      }
    });
  }

  if (mode === "operations" || mode === "architecture") {
    const addCoordinator = (kind, name, coordinator) => {
      if (!inModel(coordinator)) return;
      const model = coordinator.model || selectedModel || "application";
      const id = `${kind}:${model}:${name}`;
      add(id,kind,name,coordinator,{routePath:coordinator.path || ""});
      for (const producer of coordinator.producers || []) {
        const target = producerIDs.get(producer);
        if (target) edges.push({from:id,to:target,type:"activates"});
      }
    };
    for (const [name,route] of Object.entries(structure.coordinators?.routes || {})) addCoordinator("route",name,route);
    for (const [name,schedule] of Object.entries(structure.coordinators?.schedules || {})) addCoordinator("schedule",name,schedule);
    for (const bootstrap of structure.coordinators?.bootstrap || []) addCoordinator("bootstrap",bootstrap.name,bootstrap);

    for (const dependency of Object.values(structure.coordinators?.dependencies || {})) {
      if (!inModel(dependency)) continue;
      const model = modelName(dependency.model);
      for (const resource of dependency.dependencies || []) {
        if (resource.kind !== "state" && resource.kind !== "share") continue;
        const id = `${resource.kind}:${model}:${resource.key}`;
        add(id,resource.kind,resource.key,{...resource,model,coordinator:dependency.name});
        for (const producer of resource.producers || []) {
          const target = producerIDs.get(producer);
          if (target) edges.push({from:id,to:target,type:"dependency"});
        }
      }
    }
  }

  let center = "";
  let depth = 1;
  let viewFilter = "all";
  const graph = root.querySelector("mate-d2");
  const depthLabel = root.querySelector("[data-depth]");
  const valueDialog = root.querySelector("[data-value-dialog]");
  const valueBody = root.querySelector("[data-value-body]");
  root.querySelectorAll("[data-value-close]").forEach(button => button.onclick = () => valueDialog.close());
  valueDialog.addEventListener("click",event => {
    if (event.target === valueDialog) valueDialog.close();
  });
  const quote = value => JSON.stringify(String(value));
  const valueType = value => value === null ? "null" : Array.isArray(value) ? "array" : typeof value;
  function valueExpression(value, label = "value", open = true) {
    if (value === null || typeof value !== "object") {
      const row = document.createElement("div");
      row.className = "value-row";
      const key = document.createElement("span");
      key.className = "value-key";
      key.textContent = label;
      const code = document.createElement("code");
      code.textContent = typeof value === "string" ? JSON.stringify(value) : String(value);
      row.append(key,code);
      return row;
    }
    const details = document.createElement("details");
    details.open = open;
    const summary = document.createElement("summary");
    const entries = Array.isArray(value) ? value.map((item,index) => [String(index),item]) : Object.entries(value);
    summary.textContent = `${label} · ${Array.isArray(value) ? "Array" : "Object"} (${entries.length})`;
    details.append(summary);
    for (const [key,item] of entries) details.append(valueExpression(item,key,false));
    return details;
  }
  async function showValueDialog(node,value,eventList=events) {
    root.querySelector("[data-value-kind]").textContent = node.type.toUpperCase();
    root.querySelector("[data-value-title]").textContent = node.label;
    valueBody.replaceChildren();
    if (value === undefined) {
      const empty = document.createElement("p");
      empty.className = "value-empty";
      empty.textContent = "This value is not currently set.";
      valueBody.append(empty);
    } else {
      const expression = document.createElement("div");
      expression.className = "value-expression";
      expression.append(valueExpression(value,`${node.label} · ${valueType(value)}`));
      valueBody.append(expression);
    }
    const dependencies = dependencyInfo.get(node.id) || [];
    if (dependencies.length) {
      const list = document.createElement("details");
      list.className = "value-expression";
      const summary = document.createElement("summary");
      summary.textContent = `Dependencies (${dependencies.length})`;
      list.append(summary);
      for (const dependency of dependencies) {
        const event = eventList.filter(item => item && item.coordinator_type === "dependencies" &&
          item.coordinator_name === dependency.coordinator && item.source_kind === node.type &&
          item.source_key === node.label).slice(-1)[0];
        const item = document.createElement("details");
        const itemSummary = document.createElement("summary");
        itemSummary.textContent = dependency.coordinator;
        item.append(itemSummary);
        item.append(valueExpression(event || {}, "Latest event", true));
        item.append(valueExpression(dependency.definition || {}, "Coordinator definition", true));
        list.append(item);
      }
      valueBody.append(list);
    }
    if (!valueDialog.open && typeof valueDialog.showModal === "function") valueDialog.showModal();
    else if (!valueDialog.open) valueDialog.setAttribute("open","");
  }
  function latestCoordinatorEvent(node) {
    const candidates = events.filter(event => {
      if (!event || typeof event !== "object") return false;
      if (node.type === "route") return event.coordinator_type === "route" && event.coordinator_name === node.label;
      if (node.type === "schedule") return event.coordinator_type === "scheduler" && event.coordinator_name === node.label;
      if (node.type === "bootstrap") return event.coordinator_type === "startup" && event.coordinator_name === node.label;
      return false;
    });
    return candidates[candidates.length - 1] || null;
  }
  async function showCoordinatorDialog(node, value) {
    root.querySelector("[data-value-kind]").textContent = node.type.toUpperCase();
    root.querySelector("[data-value-title]").textContent = node.label;
    valueBody.replaceChildren();
    let eventList = events;
    try {
      const query = new URLSearchParams({id:node.id,type:node.type,model:value?.model || selectedModel});
      const response = await fetch(`/arch/detail?${query}`,{credentials:"same-origin"});
      if (response.ok) {
        const payload = await response.json();
        if (Array.isArray(payload.events)) eventList = payload.events;
      }
    } catch (_) {}
    const event = eventList.filter(item => {
      if (!item || typeof item !== "object") return false;
      if (node.type === "route") return item.coordinator_type === "route" && item.coordinator_name === node.label;
      if (node.type === "schedule") return item.coordinator_type === "scheduler" && item.coordinator_name === node.label;
      if (node.type === "bootstrap") return item.coordinator_type === "startup" && item.coordinator_name === node.label;
      return false;
    }).slice(-1)[0] || null;
    valueBody.append(valueExpression(event || {}, "Latest event", true));
    valueBody.append(valueExpression(value || {}, "Coordinator definition", true));
    if (!valueDialog.open && typeof valueDialog.showModal === "function") valueDialog.showModal();
    else if (!valueDialog.open) valueDialog.setAttribute("open","");
  }
  function showProducerDialog(node,activities) {
    root.querySelector("[data-value-kind]").textContent = "PRODUCER";
    root.querySelector("[data-value-title]").textContent = node.label;
    valueBody.replaceChildren();
    for (const activity of activities) {
      const section = document.createElement("section");
      section.className = "activity-value";
      const heading = document.createElement("h3");
      heading.textContent = `activity.${activity.name}.value`;
      section.append(heading);
      if (!activity.hasValue) {
        const empty = document.createElement("p");
        empty.className = "value-empty";
        empty.textContent = "This activity has no current value.";
        section.append(empty);
      } else {
        const expression = document.createElement("div");
        expression.className = "value-expression";
        expression.append(valueExpression(activity.value,`activity.${activity.name}.value · ${valueType(activity.value)}`));
        section.append(expression);
      }
      valueBody.append(section);
    }
    if (!activities.length) {
      const empty = document.createElement("p");
      empty.className = "value-empty";
      empty.textContent = "This producer has no activities.";
      valueBody.append(empty);
    }
    if (!valueDialog.open && typeof valueDialog.showModal === "function") valueDialog.showModal();
    else if (!valueDialog.open) valueDialog.setAttribute("open","");
  }
  const nodeSource = node => {
    const coordinator = ["route","schedule","bootstrap"].includes(node.type);
    const fill = node.type === "producer" ? (node.io ? "#fff7ed" : "#ffffff") :
      node.type === "html_activity" ? "#f8fafc" :
      (node.type === "state" || node.type === "share") ? "#ffffff" :
      coordinator ? "#f1f6f9" : "#f8fafc";
    const stroke = node.type === "producer" ? (node.io ? "#c2410c" : "#334155") :
      (node.type === "state" || node.type === "share") ? "#334155" :
      coordinator ? "#55758b" : "#64748b";
    const strokeWidth = node.io ? 4 : (node.type === "producer" ? 3 : 2);
    const profileLabel = mode === "architecture" && node.profiles?.length ? `  ·  profiles: ${node.profiles.join(", ")}` : "";
    const label = node.label + (node.io ? "  ·  IO" : "") + profileLabel;
    return `${quote(node.id)}: ${quote(label)} {
  ${coordinator ? "shape: diamond" : ""}
  icon: ${quote(iconURI(node.io ? "io" : node.type))}
  style: {
    fill: ${quote(fill)}
    stroke: ${quote(stroke)}
    stroke-width: ${strokeWidth}
    border-radius: 10
    shadow: true
  }
}`;
  };
  const edgeSource = edge => edge.type === "read" ?
    `${quote(edge.from)} -> ${quote(edge.to)} {
  style: {
    stroke: "#334155"
    stroke-width: 2
  }
  target-arrowhead.style.filled: false
}` : edge.type === "write" ?
    `${quote(edge.from)} -> ${quote(edge.to)} {
  style: {
    stroke: "#334155"
    stroke-width: 2
  }
}` : edge.type === "contains" ?
    `${quote(edge.from)} <- ${quote(edge.to)} {
  style: {
    stroke: "#334155"
    stroke-width: 2
  }
  source-arrowhead: {
    shape: diamond
    style.filled: true
  }
}` : edge.type === "dependency" ?
    `${quote(edge.from)} -> ${quote(edge.to)}: dependency {
  style: {
    stroke: "#789070"
    stroke-width: 2
    stroke-dash: 1
  }
  target-arrowhead: {
    shape: circle
    style.filled: false
  }
}` : nodes.find(node => node.id === edge.to)?.io ?
    `${quote(edge.from)} -> ${quote(edge.to)} {
  style: {
    stroke: "#c2410c"
    stroke-width: 4
  }
}` :
    `${quote(edge.from)} -> ${quote(edge.to)} {
  style: {
    stroke: "#334155"
    stroke-width: 2
  }
}`;
  function projection() {
    const uiProducers = new Set(nodes.filter(node => node.type === "producer" &&
      nodes.some(activity => activity.type === "html_activity" && activity.id.startsWith(`activity:${node.id.slice("producer:".length)}:`)))
      .map(node => node.id));
    const uiActivities = new Set(edges.filter(edge => edge.type === "contains" && uiProducers.has(edge.from)).map(edge => edge.to));
    const uiRoutes = new Set(edges.filter(edge => edge.type === "activates" && uiProducers.has(edge.to) &&
      nodes.some(node => node.id === edge.from && node.type === "route")).map(edge => edge.from));
    let filteredNodes = nodes;
    let filteredEdges = edges;
    if (viewFilter === "no-ui") {
      const excluded = new Set([...uiProducers,...uiActivities,...uiRoutes]);
      filteredEdges = edges.filter(edge => !excluded.has(edge.from) && !excluded.has(edge.to));
      const connected = new Set(filteredEdges.flatMap(edge => [edge.from,edge.to]));
      filteredNodes = nodes.filter(node => !excluded.has(node.id) &&
        (!["state","share"].includes(node.type) || connected.has(node.id)));
    } else if (viewFilter === "only-ui") {
      const included = new Set([...uiProducers,...uiActivities,...uiRoutes]);
      for (const edge of edges) {
        if (!["read","write","dependency"].includes(edge.type)) continue;
        const from = nodes.find(node => node.id === edge.from);
        const to = nodes.find(node => node.id === edge.to);
        if (included.has(edge.from) && ["state","share"].includes(to?.type)) included.add(edge.to);
        if (included.has(edge.to) && ["state","share"].includes(from?.type)) included.add(edge.from);
      }
      filteredNodes = nodes.filter(node => included.has(node.id));
      filteredEdges = edges.filter(edge => included.has(edge.from) && included.has(edge.to));
    }
    if (center && !filteredNodes.some(node => node.id === center)) center = "";
    const visible = new Set(center ? [center] : filteredNodes.map(node => node.id));
    if (center) {
      for (let level=0; level<depth; level++) {
        for (const edge of filteredEdges) {
          if (visible.has(edge.from) || visible.has(edge.to)) {
            visible.add(edge.from);
            visible.add(edge.to);
          }
        }
      }
    }
    const selectedNodes = filteredNodes.filter(node => visible.has(node.id));
    const selectedEdges = filteredEdges.filter(edge => visible.has(edge.from) && visible.has(edge.to));
    const source = [
      "direction: right",
      'style.fill: "#fbfcfe"',
      ...selectedNodes.map(nodeSource),
      ...selectedEdges.map(edgeSource)
    ].join("\n");
    return {nodes:selectedNodes,edges:selectedEdges,source};
  }
  function render() {
    graph.graph = {...projection(),interactive:mode === "operations"};
    depthLabel.textContent = center ? `${depth} relationship level${depth === 1 ? "" : "s"}` :
      (mode === "operations" ? "Complete operational projection" : "Complete declared projection");
  }
  graph.addEventListener("node-select", async event => {
    if (mode !== "operations") return;
    const selectedID = event.detail.id;
    const value = details.get(selectedID);
    const parts = selectedID.split(":");
    const selectedNode = nodes.find(node => node.id === selectedID);
    if (mode === "operations" && ["route","schedule","bootstrap"].includes(selectedNode?.type)) {
      showCoordinatorDialog(selectedNode,value);
      return;
    }
    if (selectedNode?.type === "state" || selectedNode?.type === "share") {
      root.querySelector("[data-value-kind]").textContent = selectedNode.type.toUpperCase();
      root.querySelector("[data-value-title]").textContent = selectedNode.label;
      valueBody.innerHTML = '<p class="value-empty">Loading current value…</p>';
      if (!valueDialog.open && typeof valueDialog.showModal === "function") valueDialog.showModal();
      else if (!valueDialog.open) valueDialog.setAttribute("open","");
      try {
        const query = new URLSearchParams({id:selectedID,type:selectedNode.type,model:value?.model || ""});
        const response = await fetch(`/arch/detail?${query}`,{credentials:"same-origin"});
        if (!response.ok) throw new Error("value request failed");
        const payload = await response.json();
        const entry = (payload[selectedNode.type] || []).find(item => item.key === selectedNode.label);
        showValueDialog(selectedNode,entry?.value,payload.events || events);
      } catch {
        valueBody.innerHTML = '<p class="value-empty">The current value could not be loaded.</p>';
      }
      return;
    }
    if (selectedNode?.type === "producer") {
      root.querySelector("[data-value-kind]").textContent = "PRODUCER";
      root.querySelector("[data-value-title]").textContent = selectedNode.label;
      valueBody.innerHTML = '<p class="value-empty">Loading activity values…</p>';
      if (!valueDialog.open && typeof valueDialog.showModal === "function") valueDialog.showModal();
      else if (!valueDialog.open) valueDialog.setAttribute("open","");
      try {
        const activityValues = await Promise.all((value?.activities || []).map(async activity => {
          const query = new URLSearchParams({
            id:`activity:${value.model || ""}:${selectedNode.label}:${activity.name}`,
            type:"activity",
            model:value.model || "",
            producer:selectedNode.label,
            activity:activity.name
          });
          const response = await fetch(`/arch/detail?${query}`,{credentials:"same-origin"});
          if (!response.ok) throw new Error("activity value request failed");
          const payload = await response.json();
          return {
            name:activity.name,
            hasValue:payload.execution?.has_outcome === true,
            value:payload.execution?.outcome
          };
        }));
        showProducerDialog(selectedNode,activityValues);
      } catch {
        valueBody.innerHTML = '<p class="value-empty">The activity values could not be loaded.</p>';
      }
      return;
    }
    center = selectedID;
    depth = 1;
    render();
  });
  const eventCollectionToggle = root.querySelector("[data-event-collection-toggle]");
  if (eventCollectionToggle) {
    eventCollectionToggle.addEventListener("click", async () => {
      const enabled = eventCollectionToggle.dataset.enabled === "true";
      eventCollectionToggle.disabled = true;
      try {
        const body = new URLSearchParams({
          model: eventCollectionToggle.dataset.model || selectedModel,
          enabled: String(!enabled),
          csrf_token: eventCollectionToggle.dataset.csrf || ""
        });
        const response = await fetch("/arch/event-collection", {
          method: "POST",
          credentials: "same-origin",
          headers: {"Content-Type": "application/x-www-form-urlencoded"},
          body
        });
        if (!response.ok) throw new Error("event collection update failed");
        window.location.reload();
      } catch (_) {
        eventCollectionToggle.disabled = false;
      }
    });
  }
  root.querySelector("[data-expand]").onclick = () => { depth++; render(); };
  root.querySelector("[data-collapse]").onclick = () => { depth=Math.max(0,depth-1); render(); };
  root.querySelector("[data-view-filter]")?.addEventListener("change",event => {
    if (event.target.name !== "diagram-content") return;
    viewFilter = event.target.value;
    center = "";
    depth = 1;
    render();
  });
  render();
})();
