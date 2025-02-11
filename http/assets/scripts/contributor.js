class Contributor {
	constructor(wrapper, name, avatarURL, id) {
		this.id = id

		this.elem = this.getContributorElement()
		this.name = this.getNameElement(name)
		this.avatar = this.getAvatarElement(avatarURL)
		this.wrapper = this.getWrapperElement(wrapper)

		this.elem.ondragstart = function() {
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
		div.classList.add('inline-flex', 'center-h', 'gap')
		return div
	}

	getWrapperElement(wrapper) {
		this.elem.append(this.avatar, this.name)
		wrapper.append(this.elem)
		return wrapper
	}
}
