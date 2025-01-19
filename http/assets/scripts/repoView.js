const repoInfo = document.getElementById('repo-info')
const repoID = repoInfo.dataset.repoId
const contributorID = repoInfo.dataset.contributorId
const isAdmin = repoInfo.dataset.isAdmin

class Task {
	constructor(wrapper, description, id, isCompleted) {
		this.id = id

		this.parent = this.getParentElement()
		this.remove = this.getRemoveElement()
		this.checkBox = this.getCheckBoxElement()
		this.checkBoxIcon = this.getCheckBoxIconElement()
		this.description = this.getDescriptionElement(description)
		this.checkBox.checked = isCompleted

		this.wrapper = this.getWrapper(wrapper)

		this.checkBox.onchange = (event) => this.onCheckBox(event)
		this.remove.onclick = (event) => this.onRemove(event)
	}

	getParentElement() {
		let label = document.createElement('label')
		label.classList.add('task', 'checkbox', 'flex', 'center-h', 'gap')
		label.setAttribute('task-id', this.id)
		return label
	}

	getCheckBoxElement() {
		let input = document.createElement('input')
		input.type = 'checkbox'
		return input
	}

	getCheckBoxIconElement() {
		let span = document.createElement('span')
		span.className = 'checkbox__icon'
		return span
	}

	getRemoveElement() {
		let cross = document.createElement('div')
		cross.className = 'cross'
		return cross
	}

	getDescriptionElement(value) {
		let input = document.createElement('input')
		input.className = 'description'
		input.type = 'text'
		input.value = value
		return input
	}

	getWrapper(wrapper) {
		this.parent.append(this.remove, this.checkBox, this.checkBoxIcon, this.description)
		wrapper.append(this.parent)
		return wrapper
	}

	async onRemove(event) {
		try {
			const resp = fetch('/tasks/' + this.id, {
				method: 'DELETE',
				headers: {
					'Content-type': 'application/json',
					'Accept': 'application/json',
				}
			})
			console.log(resp)
		} catch (err) {
			console.error(err)
			return false
		}
		this.wrapper.remove()
		if (titleCompleted == completedTasksList.lastElementChild) {
			completedTasksList.hidden = true
		}
	}

	onCheckBox(event) {
		if (!this.description.value.trim()) { // if its input is empty
			this.description.focus()
			return false; // prevent from adding the task to the .completed-task-list
		} else if (event.target == this.remove) {
			event.preventDefault()
			return false
		}

		if (this.wrapper.parentElement == tasksList) {
			this.description.style.opacity = '0.4'
			if (completedTasksList.hidden) {
				completedTasksList.hidden = false
			}

			completedTasksList.append(this.wrapper);

		} else {
			this.description.style.opacity = '1'
			tasksList.prepend(this.wrapper)

			if (titleCompleted == completedTasksList.lastElementChild) {
				completedTasksList.hidden = true
			}
		}
	}
}

document.addEventListener('add-task', function(event) {
	const li = document.createElement('li')
	li.classList.add('flex', 'center-h')

	if (isAdmin) {
		new Task(li, event.detail.description, event.detail.id, event.detail.isCompleted)
	} else {
		new UserTask(li, event.detail.description, event.detail.id, event.detail.isCompleted)
	}

	let target = event.target
	if (target.hidden) {
		target.hidden = false
	}
	event.target.append(li)
})

class UserTask extends Task {
	getDescriptionElement(value) {
		let p = document.createElement('p')
		p.innerHTML = value
		return p
	}

	getCheckBoxElement() {
		let input = document.createElement('input')
		input.type = 'checkbox'
		input.onclick = (event) => {
			event.preventDefault()
		}
		return input
	}

	onCheckBox(event) {
		event.preventDefault()
		return false
	}

	onRemove(event) {
		event.preventDefault()
		return false
	}

	getWrapper(wrapper) {
		this.parent.append(this.checkBox, this.checkBoxIcon, this.description)
		wrapper.append(this.parent)
		return wrapper
	}
}

const tasksPane = document.getElementById('tasks-pane')
const tasksList = document.getElementById('tasks-list')
const completedTasksList = document.getElementById('completed-tasks-list')
const titleCompleted = document.getElementById('title-completed')
const addTaskButton = document.getElementById('add-task-button')

addTaskButton.onclick = () => {
	addTask()
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

		let description = input.value
		let id = 0
		let intRepoID = parseInt(repoID)
		if (!intRepoID) {
			return false
		}

		try {
			const resp = await fetch('/tasks', {
				method: 'POST',
				headers: {
					'Content-type': 'application/json',
					'Accept': 'application/json',
				},
				body: JSON.stringify({
					description: description,
					repoID: intRepoID,
				}),
			})
			const task = await resp.json()
			id = task.id
		} catch (err) {
			console.error(err)
			return
		}

		if (id) {
			input.remove()
			new Task(li, description, id, false)
		}
	}


}

const copyContent = async (text) => {
	try {
		await navigator.clipboard.writeText(text);
		console.log('Content copied to clipboard');
	} catch (err) {
		console.error('Failed to copy: ', err);
	}
};


function connect() {
	const socket = new ReconnectingWebSocket((location.protocol == 'https:' ? 'wss:' : 'ws:') + '//' + location.host + '/events');
	socket.addEventListenter('message', function(event) {
		const e = JSON.parse(event.data)

		switch (e.type) {
			case 'task:added':
			//const li = document.createElement('li')
			//li.classList.add('flex', 'center-h')

			//if (isAdmin) {
			//	new Task(li, e.payload.task.description, false)
			//} else {
			//	new UserTask(li, e.payload.task.description, false)
			//}
			case 'task:deleted':
				taks = document.querySelector(`.task[task-id=${e.payload.id}]`)
				if (task) {
					task.remove()
				}
		}
	})
}

connect()
