const repoInfo = document.getElementById('repo-info')
const repoID = repoInfo.dataset.repoId
const contributorID = repoInfo.dataset.contributorId
const isAdmin = repoInfo.dataset.isAdmin
const tasksPane = document.getElementById('tasks-pane')
const conributorsPane = document.getElementById('contributors-pane')
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
	if (isAdmin == 'true') {
		new Task(event.detail.elem, event.detail.description, event.detail.id, event.detail.isCompleted)
	} else {
		new UserTask(event.detail.elem, event.detail.description, event.detail.id, event.detail.isCompleted)
	}

	event.detail.elem.classList.add('droppable')
	event.detail.elem.addEventListener('attach-contributor', onAttachContriubutor)

	if (event.target.hidden) {
		event.target.hidden = false
	}

	event.target.append(event.detail.elem)
})

conributorsPane.addEventListener('add-contributor', function(event) {
	const contributor = new Contributor(event.detail.elem, event.detail.name, event.detail.avatarURL, event.detail.id)

	let dropEvent = new CustomEvent('attach-contributor', {
		bubbles: true,
		detail: {
			id: event.detail.id,
			name: event.detail.name,
			avatarURL: event.detail.avatarURL,
		},

	})

	new Draggable(contributor.elem, true, dropEvent)
	event.detail.elem.classList.add('draggable')

	event.target.append(event.detail.elem)
})

function onAttachContriubutor(event) {
	if (event.target.querySelector(`.dropped[data-contributor-id="${event.detail.id}"]`)) {
		event.preventDefault()
		return false
	}

	const elem = document.createElement('div')
	elem.classList.add('dropped', 'inline-flex', 'center-h', 'gap-half')
	elem.style.padding = 0.3 + 'rem'
	elem.dataset.contributorId = event.detail.id

	let removeElem = document.createElement('div')
	removeElem.className = 'cross'
	removeElem.style.width = 1 + 'rem'
	removeElem.style.height = 1 + 'rem'
	removeElem.onclick = () => {
		elem.remove()
	}

	elem.append(removeElem, event.detail.name)
	event.target.append(elem)
}

if (isAdmin == 'true') {
	addTaskButton.onclick = () => {
		addTask()
	}
}

function addTask() {
	const input = document.createElement('input')
	tasksList.append(input)

	input.focus()
	input.onblur = async () => {
		if (!input.value.trim()) {
			li.remove()
			return
		}
		let task = await createTask(input.value)
		if (task) {
			input.remove()
			tasksList.dispatchEvent(new CustomEvent('add-task', {
				bubbles: true,
				detail: {
					elem: document.createElement('li'),
					description: task.description,
					isCompleted: false,
					id: task.id,
				}
			}))
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
				let checkbox = document.querySelector(`.task[data-task-id="${e.payload.id}"] input[type="checkbox"]`)
				if (checkbox != null) {
					checkbox.dispatchEvent(new Event('change', {
						bubbles: false,
					}))
				} else {
					console.log("couldn't find task by id: " + e.payload.id)
				}
				break
			case 'task:description_changed':
				let input = document.querySelector(`.task[data-task-id="${e.payload.id}"] input[type="text"]`)
				if (input != null) {
					input.value = e.payload.value
				} else {
					console.log("couldn't find task by id: " + e.payload.id)
				}
				break
		}
	}
}

connect()
