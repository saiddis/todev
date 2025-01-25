const repoInfo = document.getElementById('repo-info')
const repoID = repoInfo.dataset.repoId
const contributorID = repoInfo.dataset.contributorId
const isAdmin = repoInfo.dataset.isAdmin
const tasksPane = document.getElementById('tasks-pane')
const tasksList = document.getElementById('tasks-list')
const completedTasksList = document.getElementById('completed-tasks-list')
const titleCompleted = document.getElementById('title-completed')
const addTaskButton = document.getElementById('add-task-button')
const copyContent = async (text) => {
	try {
		await navigator.clipboard.writeText(text);
		console.log('Content copied to clipboard');
	} catch (err) {
		console.error('Failed to copy: ', err);
	}
}

tasksPane.addEventListener('escape-task', function(event) {
	if (event.target.parentElement == tasksList) {
		if (!event.detail.remove) {
			completedTasksList.append(event.target)
		} else {
			event.target.remove()
		}
	} else if (event.target.parentElement == completedTasksList) {
		if (!event.detail.remove) {
			tasksList.append(event.target)
		} else {
			event.target.remove()
		}
	}

	if (completedTasksList.childElementCount < 3) {
		completedTasksList.hidden = true
	} else {
		completedTasksList.hidden = false
	}
})

tasksPane.addEventListener('add-task', function(event) {
	let li = document.createElement('li')
	li.classList.add('flex', 'center-h')

	if (isAdmin == 'true') {
		new Task(li, event.detail.description, event.detail.id, event.detail.isCompleted)
	} else {
		new UserTask(li, event.detail.description, event.detail.id, event.detail.isCompleted)
	}

	if (event.target.hidden) {
		event.target.hidden = false
	}
	event.target.append(li)
})

if (isAdmin == 'true') {
	addTaskButton.onclick = () => {
		addTask()
	}
}

function addTask() {
	const li = document.createElement('li')
	li.classList.add('flex', 'center-h')
	const input = document.createElement('input')
	li.append(input)
	tasksList.append(li)

	input.focus()
	input.onblur = async () => {
		if (!input.value.trim()) {
			li.remove()
			return
		}
		let task = await createTask(input.value)
		if (task) {
			input.remove()
			new Task(li, task.description, task.id, false)
		}
	}
}

async function createTask(description) {
	try {

		const resp = await fetch('/tasks', {
			method: 'POST',
			headers: {
				'Content-type': 'application/json',
				'Accept': 'application/json',
			},
			body: JSON.stringify({
				description: description,
				repoID: parseInt(repoID),
			}),
		})

		if (resp.ok) {
			const task = await resp.json()
			return task
		}
		return null
	} catch (err) {
		console.error(err)
		return null
	}
}

function connect() {
	const socket = new ReconnectingWebSocket((location.protocol == 'https:' ? 'wss:' : 'ws:') + '//' + location.host + '/events');
	socket.onmessage = function(event) {
		const e = JSON.parse(event.data)
		switch (e.type) {
			case 'task:added':
				const li = document.createElement('li')
				li.classList.add('flex', 'center-h')

				if (isAdmin == 'true') {
					new Task(li, e.payload.task.description, e.payload.task.id, false)
				} else {
					new UserTask(li, e.payload.task.description, e.payload.task.id, false)
				}

				tasksList.append(li)
				break
			case 'task:deleted':
				let task = document.querySelector(`.task[data-task-id="${e.payload.id}"]`)
				if (task != null) {
					task.dispatchEvent(new CustomEvent('escape-task', {
						bubbles: true,
						detail: {
							remove: true,
						}
					}))
				} else {
					console.log("couldn't find task by id: " + e.payload.id)
				}
				break
			case 'task:completion_toggled':
				let input = document.querySelector(`.task[data-task-id="${e.payload.id}"] input[type="checkbox"]`)
				if (input != null) {
					input.dispatchEvent(new Event('change', {
						bubbles: false,
					}))
				} else {
					console.log("couldn't find task by id: " + e.payload.id)
				}
				break
		}
	}
}

connect()
