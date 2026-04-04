package runtime

func (js *JavaScriptRuntime) createAxiosMock() string {
	return `
		const axios = {
			get: async function(url, config = {}) {
				return {
					status: 200,
					statusText: 'OK',
					data: { success: true, url: url },
					headers: { 'content-type': 'application/json' }
				};
			},
			post: async function(url, data, config = {}) {
				return {
					status: 200,
					statusText: 'OK',
					data: { success: true, url: url, posted: data },
					headers: { 'content-type': 'application/json' }
				};
			},
			put: async function(url, data, config = {}) {
				return {
					status: 200,
					statusText: 'OK',
					data: { success: true, url: url, put: data },
					headers: { 'content-type': 'application/json' }
				};
			},
			delete: async function(url, config = {}) {
				return {
					status: 200,
					statusText: 'OK',
					data: { success: true, url: url, deleted: true },
					headers: { 'content-type': 'application/json' }
				};
			}
		};

		axios.create = function(config) {
			return axios;
		};

		module.exports = axios;
	`
}

func (js *JavaScriptRuntime) createLodashMock() string {
	return `
		const _ = {
			get: function(object, path, defaultValue) {
				const keys = path.split('.');
				let result = object;
				for (const key of keys) {
					if (result && typeof result === 'object' && key in result) {
						result = result[key];
					} else {
						return defaultValue;
					}
				}
				return result;
			},
			set: function(object, path, value) {
				const keys = path.split('.');
				let current = object;
				for (let i = 0; i < keys.length - 1; i++) {
					const key = keys[i];
					if (!(key in current) || typeof current[key] !== 'object') {
						current[key] = {};
					}
					current = current[key];
				}
				current[keys[keys.length - 1]] = value;
				return object;
			},
			clone: function(obj) {
				return JSON.parse(JSON.stringify(obj));
			},
			merge: function(target, ...sources) {
				return Object.assign(target, ...sources);
			},
			isEmpty: function(value) {
				return value == null || (typeof value === 'object' && Object.keys(value).length === 0);
			},
			isArray: Array.isArray,
			isObject: function(value) {
				return value != null && typeof value === 'object' && !Array.isArray(value);
			},
			map: function(collection, iteratee) {
				if (Array.isArray(collection)) {
					return collection.map(iteratee);
				}
				return Object.keys(collection).map(key => iteratee(collection[key], key));
			},
			filter: function(collection, predicate) {
				if (Array.isArray(collection)) {
					return collection.filter(predicate);
				}
				const result = {};
				Object.keys(collection).forEach(key => {
					if (predicate(collection[key], key)) {
						result[key] = collection[key];
					}
				});
				return result;
			}
		};

		module.exports = _;
	`
}

func (js *JavaScriptRuntime) createMomentMock() string {
	return `
		function moment(input, format) {
			const date = input ? new Date(input) : new Date();

			return {
				format: function(fmt = 'YYYY-MM-DD HH:mm:ss') {
					return date.toISOString().substring(0, 19).replace('T', ' ');
				},
				add: function(amount, unit) {
					const newDate = new Date(date);
					switch(unit) {
						case 'days': newDate.setDate(newDate.getDate() + amount); break;
						case 'hours': newDate.setHours(newDate.getHours() + amount); break;
						case 'minutes': newDate.setMinutes(newDate.getMinutes() + amount); break;
						case 'seconds': newDate.setSeconds(newDate.getSeconds() + amount); break;
					}
					return moment(newDate);
				},
				subtract: function(amount, unit) {
					return this.add(-amount, unit);
				},
				unix: function() {
					return Math.floor(date.getTime() / 1000);
				},
				valueOf: function() {
					return date.getTime();
				},
				toDate: function() {
					return new Date(date);
				},
				isValid: function() {
					return !isNaN(date.getTime());
				}
			};
		}

		moment.now = function() {
			return Date.now();
		};

		moment.utc = function(input) {
			return moment(input);
		};

		module.exports = moment;
	`
}

func (js *JavaScriptRuntime) createUuidMock() string {
	return `
		const uuid = {
			v4: function() {
				return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
					const r = Math.random() * 16 | 0;
					const v = c === 'x' ? r : (r & 0x3 | 0x8);
					return v.toString(16);
				});
			}
		};

		module.exports = uuid;
	`
}

func (js *JavaScriptRuntime) createCryptoJsMock() string {
	return `
		const CryptoJS = {
			MD5: function(message) {
				return { toString: function() { return 'mock-md5-hash'; } };
			},
			SHA1: function(message) {
				return { toString: function() { return 'mock-sha1-hash'; } };
			},
			SHA256: function(message) {
				return { toString: function() { return 'mock-sha256-hash'; } };
			},
			enc: {
				Base64: {
					stringify: function(wordArray) { return btoa('mock-data'); },
					parse: function(base64) { return { toString: function() { return 'mock-data'; } }; }
				},
				Utf8: {
					stringify: function(wordArray) { return 'mock-utf8-string'; },
					parse: function(utf8) { return { toString: function() { return 'mock-data'; } }; }
				}
			}
		};

		module.exports = CryptoJS;
	`
}
