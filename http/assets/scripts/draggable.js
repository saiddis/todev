class Draggable {
	constructor(elem, isClone, dropEvent) {
		this.elem = elem
		this.isClone = isClone
		this.dropEvent = dropEvent
		this.elem.addEventListener('mousedown', (event) => this.onMouseDown(event))
	}

	getDraggable() {
		let draggable = this.isClone ? this.elem.cloneNode(true) : this.elem
		draggable.style.position = 'absolute'
		draggable.zIndex = 1000;
		return draggable
	}

	onMouseDown(event) {
		let draggable = this.getDraggable()
		let currentDroppable = null
		document.body.append(draggable)

		let shiftX = event.clientX - this.elem.getBoundingClientRect().left
		let shiftY = event.clientY - this.elem.getBoundingClientRect().top

		function moveAt(pageX, pageY) {
			draggable.style.left = pageX - shiftX + 'px';
			draggable.style.top = pageY - shiftY + 'px';
		}

		moveAt(event.pageX, event.pageY,)


		const onMouseMove = (event) => {
			moveAt(event.pageX, event.pageY)

			draggable.style.display = 'none'
			let elemBelow = document.elementFromPoint(event.clientX, event.clientY)
			draggable.style.display = 'flex'

			if (!elemBelow) return

			let droppableBelow = elemBelow.closest('.droppable')
			if (currentDroppable != droppableBelow) {
				if (currentDroppable) {
					this.leaveDroppable(currentDroppable)
				}
				currentDroppable = droppableBelow
				if (currentDroppable) {
					this.enterDroppable(currentDroppable)
				}

			}
		}
		document.addEventListener('mousemove', onMouseMove)

		draggable.onmouseup = () => {
			document.removeEventListener('mousemove', onMouseMove)
			draggable.onmouseup = null;

			if (currentDroppable) {
				draggable.remove()
				currentDroppable.dispatchEvent(this.dropEvent)
			} else {
				draggable.remove()
			}

			this.leaveDroppable(currentDroppable)
		}

		draggable.ondragstart = function() {
			return false
		}
	}


	leaveDroppable(elem) {
		elem.style.backgroundColor = ''
	}

	enterDroppable(elem) {
		elem.style.backgroundColor = 'pink'
	}
}
