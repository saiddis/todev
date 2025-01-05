class Task {
	constructor(parent, cross, checkBox, input) {
		this.cross = cross
		this.checkBox = checkBox
		this.input = input
		this.task = parent

		this.checkBox.onchange = (event) => this.onCheckBox(event)
		this.input.onblur = () => {
			if (!this.input.value.trim()) {
				this.task.remove()
			}
		}

		this.cross.onclick = () => {
			this.task.remove()

			if (titleCompleted == completedTasksList.lastElementChild) {
				completedTasksList.hidden = true
			}
		}
	}

	onCheckBox(event) {
		if (!this.input.value.trim()) { // if its input is empty
			this.input.focus()
			return false; // prevent from adding the task to the .completed-task-list
		} else if (event.target == this.input) {
			return false
		}

		if (this.task.parentElement == tasksList) {
			this.task.style.opacity = '0.5'
			if (completedTasksList.hidden) {
				completedTasksList.hidden = false
			}

			completedTasksList.append(this.task);

		} else {
			this.task.style.opacity = '1'
			tasksList.prepend(this.task)

			if (titleCompleted == completedTasksList.lastElementChild) {
				completedTasksList.hidden = true
			}
		}
	}
}

const tasksList = document.getElementById('tasks-list')
const completedTasksList = document.getElementById('completed-tasks-list')
const titleCompleted = document.getElementById('title-completed')
const addTaskButton = document.getElementById('add-task-button')

addTaskButton.onclick = () => {
	addTask()
}

function addTask() {
	const task = `
	<label class="checkbox flex center-h gap">
	<div class="cross"></div>
	<input type="checkbox">
	<span class="checkbox__icon"></span>
	<input class="description" type="text" />
	</label>
`
	const li = document.createElement('li')
	li.classList.add('flex', 'center-h')
	li.insertAdjacentHTML('beforeend', task)
	const cross = li.firstElementChild.firstElementChild
	const checkBox = li.querySelector('.checkbox')
	const input = li.querySelector('.description')
	tasksList.append(li)

	input.focus()
	input.onblur = () => {
		if (!input.value.trim()) {
			li.remove()
		}

		new Task(li, cross, checkBox, input)
	}
}
