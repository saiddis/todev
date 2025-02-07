class Draggable {
	constructor(elem, isClone, dropCallback) {
		this.elem = elem
		this.isClone = isClone
		this.dropCallback = dropCallback

		this.elem.addEventListener('mousedown', (event) => this.onMouseDown(event))
	}

	onMouseDown(event) {
		let currentDroppable = null;
		let draggable = this.contributor.cloneNode(this.isClone)

		draggable.style.position = 'absolute'
		draggable.zIndex = 1000;

		document.body.append(draggable)

		let shiftX = event.clientX - this.contributor.getBoundingClientRect().left
		let shiftY = event.clientY - this.contributor.getBoundingClientRect().top

		moveAt(event.pageX, event.pageY)

		function moveAt(pageX, pageY) {
			draggable.style.left = pageX - shiftX + 'px';
			draggable.style.top = pageY - shiftY + 'px';
		}

		function onMouseMove(event) {
			moveAt(event.pageX, event.pageY)

			draggable.style.display = 'none'
			let elemBelow = document.elementFromPoint(event.clientX, event.clientY)
			draggable.style.display = 'flex'

			if (!elemBelow) return

			let droppableBelow = elemBelow.closest('.droppable')
			if (currentDroppable != droppableBelow) {
				if (currentDroppable) {
					leaveDroppable(currentDroppable)
				}
				currentDroppable = droppableBelow
				if (currentDroppable) {
					enterDroppable(currentDroppable)
				}

			}
		}

		document.addEventListener('mousemove', onMouseMove)

		draggable.onmouseup = () => {
			document.removeEventListener('mousemove', onMouseMove)
			draggable.onmouseup = null;
			this.onDrop()
		}

		draggable.ondragstart = function() {
			return false
		}
	}

	leaveDroppable() {
		elem.style.backgroundColor = ''
	}

	enterDroppable() {
		elem.style.backgroundColor = 'pink'
	}
}

class Contributor {
	constructor(wrapper, name, avatarURL, id) {
		this.id = id

		this.contributor = this.getContributorElement()
		this.name = this.getNameElement(name)
		this.avatar = this.getAvatarElement(avatarURL)
		this.wrapper = this.getWrapperElement(wrapper)

		this.contributor.ondragstart = function() {
			return false
		}
	}

	getNameElement(name) {
		let h3 = document.createElement('h3')
		h3.innerHTML = name
		return h3
	}

	getAvatarElement(avatarURL) {
		let img = document.createElement('img')
		img.src = avatarURL
		img.classList.add('avatar', 'small')
		return img
	}

	getContributorElement() {
		let div = document.createElement('div')
		div.classList.add('flex', 'center-h', 'gap')
		return div
	}

	getWrapperElement(wrapper) {
		this.contributor.append(this.avatar, this.name)
		wrapper.append(this.contributor)
		return wrapper
	}
}
