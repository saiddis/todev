class Task {
	constructor(wrapper, description, id, isCompleted) {
		this.id = id

		this.elem = this.getTaskElement()
		this.remove = this.getRemoveElement()
		this.checkBox = this.getCheckBoxElement()
		this.checkBoxIcon = this.getCheckBoxIconElement()
		this.description = this.getDescriptionElement(description)
		this.checkBox.checked = isCompleted

		this.wrapper = this.getWrapper(wrapper)

		this.checkBox.onchange = (event) => this.onCheckBox(event)
		this.remove.onclick = (event) => this.onRemove(event)
		this.description.onchange = (event) => this.onDescriptionChange(event)
	}

	getTaskElement() {
		let label = document.createElement('label')
		label.classList.add('checkbox', 'inline-flex', 'center-h', 'gap')
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
		this.elem.append(this.checkBox, this.checkBoxIcon, this.description)
		wrapper.className = 'task'
		wrapper.setAttribute('data-task-id', this.id)
		wrapper.append(this.remove, this.elem)
		wrapper.classList.add('flex', 'gap', 'center-h')
		return wrapper
	}


	onRemove(event) {
		if (event.target != this.remove) {
			event.preventDefault()
			return false
		}
		deleteTask(this.id).
			then(ok => {
				if (!ok) {
					event.preventDefault()
					return false
				}
			})
		this.wrapper.dispatchEvent(new CustomEvent('escape-task', {
			bubbles: true,
			detail: {
				id: this.id,
				remove: true,
			}
		}))
	}

	onDescriptionChange(event) {
		if (event.target != this.description) {
			event.preventDefault()
			return false
		} else if (!this.description.value.trim()) {
			this.description.focus()
			return false; // prevent from adding the elem to the .completed-elem-list
		}

		changeDescription(this.id, this.description.value).
			then(value => {
				if (value) {
					this.description.value = value
				} else {
					event.preventDefault()
					return false
				}
			})
	}

	onCheckBox(event) {
		if (event.target != this.checkBox) {
			event.preventDefault()
			return false
		} else if (!this.description.value.trim()) { // if its input is empty
			this.description.focus()
			return false; // prevent from adding the elem to the .completed-elem-list
		}

		toggleCompletion(this.id).
			then(ok => {
				if (!ok) {
					event.preventDefault()
					return false
				}
			})

		this.wrapper.dispatchEvent(new CustomEvent('escape-task', {
			bubbles: true,
			detail: {
				id: this.id,
				remove: false,
			}
		}))
		if (this.description.style.opacity = '0.4') {

			this.description.style.opacity = '1'
		} else {
			this.description.style.opacity = '0.4'
		}
	}
}


class UserTask extends Task {
	getDescriptionElement(value) {
		let input = document.createElement('input')
		input.className = 'description'
		input.type = 'text'
		input.value = value
		input.readOnly = true
		return input
	}

	getCheckBoxElement() {
		let input = document.createElement('input')
		input.type = 'checkbox'
		input.onclick = (event) => {
			event.preventDefault()
		}
		return input
	}

	getWrapper(wrapper) {
		this.elem.append(this.checkBox, this.checkBoxIcon, this.description)
		wrapper.className = 'task'
		wrapper.setAttribute('data-task-id', this.id)
		wrapper.append(this.elem)
		wrapper.classList.add('flex', 'gap', 'center-h')
		return wrapper
	}

	onCheckBox() {
		this.wrapper.dispatchEvent(new CustomEvent('escape-task', {
			bubbles: true,
			detail: {
				id: this.id,
				remove: false,
			}
		}))

		this.checkBox.checked = this.checkBox.checked ? false : true

		if (this.description.style.opacity = '0.4') {
			this.description.style.opacity = '1'
		} else {
			this.description.style.opacity = '0.4'
		}
	}

}

async function deleteTask(taskId) {
	try {

		let resp = await fetch('/tasks/' + taskId, {
			method: 'DELETE',
			headers: {
				'Content-type': 'application/json',
				'Accept': 'application/json',
			}
		})

		if (resp.ok) {
			return true
		}
		return false
	} catch (err) {
		console.error(err)
		return false
	}
}

async function toggleCompletion(taskId) {
	try {

		let resp = await fetch('/tasks/' + taskId, {
			method: 'PATCH',
			headers: {
				'Content-Type': 'application/json',
				'Accept': 'application/json',
			},
			body: JSON.stringify({
				toggleCompletion: true,
			}),
		})
		if (resp.ok) {
			return true
		}
		return false
	} catch (err) {
		console.error(err)
		return false
	}
}

async function changeDescription(elemID, value) {
	try {
		let resp = await fetch('/tasks/' + elemID, {
			method: 'PATCH',
			headers: {
				'Content-Type': 'application/json',
				'Accept': 'application/json',
			},
			body: JSON.stringify({
				description: value,
			}),
		})
		if (resp.ok) {
			let elem = resp.json()
			return elem.description
		}
		return ""
	} catch (err) {
		console.error(err)
		return ""
	}
}
