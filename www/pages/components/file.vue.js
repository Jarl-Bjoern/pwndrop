var appFile = Vue.component("app-file", {
    template: `
	<div ref="uploadItem" class="upload-item" :class="{'upload-disabled': !file.is_enabled, 'upload-paused': file.is_paused}">

                <!-- Download log modal -->
		<b-modal
			:id="'log-modal-' + file.id"
			size="lg"
			hide-header
			ok-only
			ok-title="Close"
			@show="startLogRefresh()"
			@hide="stopLogRefresh()"
		>
			<div class="d-flex justify-content-between align-items-center" style="margin-bottom:8px;">
				<strong><i class="fas fa-history" style="margin-right:6px"></i>Download log: {{ file.name }}</strong>
				<div>
					<small class="text-muted" style="margin-right:10px;">auto-refresh every 5s</small>
					<button class="btn btn-sm btn-outline-danger" @click="clearLog()" v-tooltip:top="'Clear log'">
						<i class="fas fa-trash"></i>
					</button>
				</div>
			</div>
			<div v-if="log_entries.length === 0" class="text-muted text-center" style="padding: 20px 0;">
				No downloads yet
			</div>
			<table v-else class="table table-sm table-dark" style="font-size:12px; margin-bottom:0;">
				<thead>
					<tr>
						<th style="width:160px">Date / Time</th>
						<th style="width:130px">IP</th>
						<th>User-Agent</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="entry in log_entries" :key="entry.id">
						<td>{{ entry.download_time | formatTime }}</td>
						<td>{{ entry.ip }}</td>
						<td style="word-break:break-all; font-size:11px;">{{ entry.user_agent }}</td>
					</tr>
				</tbody>
			</table>
		</b-modal>

		<div class="row upload-desc">
			<div class="settings">
                <span class="btn-col">
				<button class="btn btn-sm btn-primary btn-circle-sm" @click="$emit('editFile', file.id)" v-tooltip:top="'Change file settings'">
					<i class="fas fa-cog"></i>
				</button>
                </span>
                <span class="btn-col">
				<button class="btn btn-sm btn-circle-sm" :class="{'btn-outline-secondary': !file.is_enabled, 'btn-success': file.is_enabled}" @click="$emit('enableFile', file.id)" v-tooltip:top="'Make file available for download'">
					<i class="fas fa-power-off"></i>
				</button>
                </span>
                <span class="btn-col">
				<button class="btn btn-sm btn-circle-sm" :class="{'btn-outline-secondary': !file.is_paused, 'btn-gray': file.is_paused}" @click="$emit('pauseFile', file.id)" v-tooltip:top="'Enable the facade and serve the facade file instead of the original one'">
					<i class="fas fa-mask"></i>
				</button>
                </span>
                <span class="btn-col">
				<button class="btn btn-sm btn-outline-secondary btn-circle-sm" @click="openLog()" v-tooltip:top="'Download log'">
					<i class="fas fa-history"></i>
				</button>
                </span>
			</div>
			<div class="col clip trans" :class="{'text-dim': file.is_paused}">
				<span class="title">{{ file.name }}</span>
			</div>
            <div v-if="file.sub_file != null && !file.is_paused" class="col-auto shrink">
                <i class="fas fa-arrow-left"></i>
            </div>
            <div v-else-if="file.sub_file != null" class="col-auto shrink">
                <i class="fas fa-arrow-right"></i>
            </div>
			<div v-if="file.sub_file != null" class="col clip trans" :class="{'text-dim': !file.is_paused}">
				<span class="title">{{ file.sub_name }}</span>
			</div>
			<div v-if="folderName" class="col-auto shrink d-none d-sm-block">
				<span class="file-folder-tag" @click.stop="$emit('filterByFolder', file.folder_id)" :title="'Filter by: ' + folderName">
					<i class="fas fa-folder" style="margin-right:3px;font-size:10px;"></i>{{ folderName }}
				</span>
			</div>
			<div class="d-none d-sm-block col-auto shrink text-right clip">
				<span class="fsize">{{ file.fsize | prettyBytes }}</span>
			</div>
			<div class="controls">
				<button class="btn btn-sm btn-danger btn-circle-sm" @click="deleteItem(file.id)">
					<i class="fas fa-times"></i>
				</button>
			</div>
		</div>
		<div class="row upload-info" v-show="!file.progress || file.progress == 100">
			<div class="col-auto shrink clip">
				<span class="btn-col">
					<a class="btn-copy" ref="copyUrl" href @click.prevent="copyHttpUrl()">
						<button class="btn btn-sm btn-outline-success btn-copy-link" v-tooltip:bottom="'Copy HTTP link to clipboard'">
							<i class="fas fa-copy" style="margin-right: 5px"></i>HTTP
						</button>
					</a>
				</span>
				<span class="btn-col">
					<a class="btn-copy" ref="copyWebdavUrl" href @click.prevent="copyWebdavUrl()">
						<button class="btn btn-sm btn-outline-success btn-copy-link" v-tooltip:bottom="'Copy WebDAV link to clipboard'">
							<i class="fas fa-copy" style="margin-right: 5px"></i>WebDAV
						</button>
					</a>
				</span>
			</div>
            <div class="col-auto grow trans" :class="{'text-lg': file.is_paused}">
                <small>{{ file.url_path }}</small>
            </div>
			<div class="d-none d-sm-block col text-right clip trans" :class="{'text-lg': file.is_paused}">
				<small>{{ file.mime_type }}</small>
			</div>
		</div>
		<div class="row upload-meta" v-if="(file.comment && file.comment !== '') || (file.max_downloads > 0)" v-show="!file.progress || file.progress == 100">
			<div class="col" style="padding-top: 3px; padding-bottom: 3px;">
				<small v-if="file.comment && file.comment !== ''" class="text-muted" style="margin-right: 12px;">
					<i class="fas fa-comment-alt" style="margin-right: 4px;"></i>{{ file.comment }}
				</small>
				<small v-if="file.max_downloads > 0" class="text-muted">
					<i class="fas fa-download" style="margin-right: 4px;"></i>{{ file.downloads_left || 0 }} left of {{ file.max_downloads }}
				</small>
			</div>
		</div>
		<div class="row">
			<div class="file-progress col" v-if="file.progress < 100">
				<div class="progress">
					<div
						class="progress-bar progress-bar-striped progress-bar-animated bg-success"
						role="progressbar"
						:style="{width: file.progress+'%'}"
						aria-valuemin="0"
						aria-valuemax="100"
						:aria-valuenow="file.progress"
					></div>
				</div>
			</div>
		</div>
	</div>
    `,
    $_veeValidate: {
        validator: "new"
    },
//	props: ["file"],
	props: ["file", "folderName"],
	data() {
		return {
			log_entries: [],
			log_timer: null,
			api_url: Config.Hostname + Config.AdminDir + "/" + Config.ApiPath
		};
	},
	filters: {
		formatTime(unix) {
			if (!unix) return "";
			var d = new Date(unix * 1000);
			var pad = n => String(n).padStart(2, "0");
			return d.getFullYear() + "-" + pad(d.getMonth()+1) + "-" + pad(d.getDate()) +
				" " + pad(d.getHours()) + ":" + pad(d.getMinutes()) + ":" + pad(d.getSeconds());
		}
	},
	methods: {
		copyHttpUrl() {
			var l = window.location;
			var url = l.protocol + "//" + l.hostname;
			if (l.port != "" && (l.port != 443 && l.port != 80)) {
				url += ":" + l.port;
			}
			url += escape(this.file.url_path);
			this.$refs.copyUrl.setAttribute("data-clipboard-text", url);
		},
		copyWebdavUrl() {
			var l = window.location;
			var url = "\\\\" + l.hostname + "@80";
			/*if (l.port != 443 && l.port != 80) {
				url += "@" + l.port;
			}*/
			url += escape(this.file.url_path).replace(/\//g, "\\");
			this.$refs.copyWebdavUrl.setAttribute("data-clipboard-text", url);
		},
		deleteItem(id) {
			this.$refs.uploadItem.style.width =
				this.$refs.uploadItem.offsetWidth + "px";
			this.$emit("deleteFile", id);
		},

		openLog() {
			this.fetchLog();
			this.$bvModal.show("log-modal-" + this.file.id);
		},
		fetchLog() {
			var vm = this;
			axios.get(vm.api_url + "/files/" + vm.file.id + "/log")
				.then(response => {
					vm.log_entries = response.data.data.logs || [];
				})
				.catch(error => console.log(error));
		},
		startLogRefresh() {
			this.fetchLog();
			this.log_timer = setInterval(() => { this.fetchLog(); }, 5000);
		},
		stopLogRefresh() {
			if (this.log_timer) {
				clearInterval(this.log_timer);
				this.log_timer = null;
			}
		},
		clearLog() {
			var vm = this;
			axios.delete(vm.api_url + "/files/" + vm.file.id + "/log")
				.then(() => { vm.log_entries = []; })
				.catch(error => console.log(error));
		}
	},
	beforeDestroy() {
		this.stopLogRefresh();
	}
})
