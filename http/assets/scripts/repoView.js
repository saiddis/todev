class Task {
	constructor(parent, cross, checkBox, input) {
		this.cross = cross
		this.checkBox = checkBox
		this.input = input
		this.task = parent

		this.onCheckBox = this.onCheckBox.bind(this)
		this.checkBox.onclick = () => this.onCheckBox()
		this.input.onblur = () => {
			if (!this.input.value.trim()) {
				this.task.remove()
			}
		}
		this.cross.onclick = () => {
			this.task.remove()
		}
	}

	onCheckBox() {
		if (!this.input.value.trim()) { // if its input is empty

			this.checkBox.nextElementSibling.focus();
			return false; // prevent from adding the task to the .completed-task-list
		}

		if (this.checkBox.lastElementChild.style.display == 'none') {
			this.checkBox.lastElementChild.style.display = ''
			this.checkBox.nextSibling.style.opacity = '0.3'
			if (completedTasksList.hidden) {
				completedTasksList.hidden = false
			}

			completedTasksList.append(this.task);

		} else {
			this.checkBox.lastElementChild.style.display = 'none'
			this.checkBox.nextSibling.style.opacity = '1'
			tasksList.prepend(this.task)
			if (titleCompleted == completedTasksList.lastElementChild) {
				completedTasksList.hidden = true
			}
		}
	}
}

const crossIcon = `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g id="SVGRepo_bgCarrier" stroke-width="0"></g><g id="SVGRepo_tracerCarrier" stroke-linecap="round" stroke-linejoin="round"></g><g id="SVGRepo_iconCarrier"> <path d="M6.99486 7.00636C6.60433 7.39689 6.60433 8.03005 6.99486 8.42058L10.58 12.0057L6.99486 15.5909C6.60433 15.9814 6.60433 16.6146 6.99486 17.0051C7.38538 17.3956 8.01855 17.3956 8.40907 17.0051L11.9942 13.4199L15.5794 17.0051C15.9699 17.3956 16.6031 17.3956 16.9936 17.0051C17.3841 16.6146 17.3841 15.9814 16.9936 15.5909L13.4084 12.0057L16.9936 8.42059C17.3841 8.03007 17.3841 7.3969 16.9936 7.00638C16.603 6.61585 15.9699 6.61585 15.5794 7.00638L11.9942 10.5915L8.40907 7.00636C8.01855 6.61584 7.38538 6.61584 6.99486 7.00636Z" fill="#0F0F0F" style="--darkreader-inline-fill: #0b0c0d;" data-darkreader-inline-fill=""></path> </g></svg>`
const checkBoxIcon = `
        <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" fill="#000000" style="--darkreader-inline-fill: var(--darkreader-background-000000, #000000);" data-darkreader-inline-fill=""><g id="SVGRepo_bgCarrier" stroke-width="0"></g><g id="SVGRepo_tracerCarrier" stroke-linecap="round" stroke-linejoin="round"></g><g id="SVGRepo_iconCarrier"> <rect fill="#f9fbff" stroke-width="1" stroke="#868c8f" x="0.5" y="0.5" width="23" height="23" rx="5.5" style="--darkreader-inline-fill: #1a1c1d; --darkreader-inline-stroke: #9e9689;" data-darkreader-inline-fill="" data-darkreader-inline-stroke=""></rect> <path fill="#494c4e" d="M19.707,7.293a1,1,0,0,0-1.414,0L10,15.586,6.707,12.293a1,1,0,0,0-1.414,1.414l4,4a1,1,0,0,0,1.414,0l9-9A1,1,0,0,0,19.707,7.293Z" style="--darkreader-inline-fill: #393e40;" data-darkreader-inline-fill=""></path> </g></svg>`

const tasksList = document.getElementById('tasks-list')
const completedTasksList = document.getElementById('completed-tasks-list')
const titleCompleted = document.getElementById('title-completed')
const addTaskButton = document.getElementById('add-task-button')

addTaskButton.onclick = () => {
	addTask()
}

function addTask() {
	const li = document.createElement('li')
	const input = document.createElement('input')
	li.classList.add('flex', 'center-v')
	li.insertAdjacentHTML('afterbegin', crossIcon)
	li.insertAdjacentHTML('beforeend', checkBoxIcon)
	const cross = li.firstElementChild
	const checkBox = cross.nextElementSibling
	checkBox.lastElementChild.lastElementChild.style.display = 'none'

	li.append(input);
	tasksList.append(li)

	input.focus()
	input.onblur = () => {
		if (!input.value.trim()) {
			li.remove()
		}

		new Task(li, cross, checkBox, input)
	}
}
