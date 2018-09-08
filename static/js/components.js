// Tags in Search bar
Vue.component("search-tag", {
    props: ["name", "color", "show"],
    template: `
	<div @mouseenter="show = true;" @mouseleave="show = false;" :style="{ 'background-color': color }" class="tag">
		<div>{{name}}</div>
		<i v-show="show" @click="deleteTagFromSearch(name);" class="material-icons" style="cursor: pointer; font-size: 20px;">close</i>
	</div>`,
    methods: {
        deleteTagFromSearch: function(name) {
            this.$parent.input().tags.delete(name);
        }
    }
});

// Tags in Main block
Vue.component("file-tag", {
    props: ["tag"],
    template: `
	<div :style="{ 'background-color': tag.color }" class="tag">
		<div>{{tag.name}}</div>
	</div>`
});

// For drag and drop input
Vue.component("tags-input", {
    props: ["name", "color"],
    methods: {
        startDrag: function(ev) {
            // Sometimes there's a bug, when user drag text, not div, so we need to check nodeName
            // If nodeName == "#text", user dragged text. We still can drop tag, but there's some graphic artifacts
            if (ev.target.nodeName == "DIV") {
                ev.dataTransfer.setData(
                    "tagName",
                    ev.target.children[0].textContent
                );
            } else if (ev.target.nodeName == "#text") {
                ev.dataTransfer.setData("tagName", ev.target.data);
            } else {
                console.error("Error: can't get the name of a tag");
            }
        }
    },
    template: `
	<div :style="{ 'background-color': color }" class="tag vertically" style="margin-bottom: 5px; margin-top: 5px;" draggable="true" @dragstart="startDrag">
		<div>{{name}}</div>
	</div>`
});

const validTagName = /^[\w\d- ]*$/;
const validColor = /^#[\dabcdef]{6}$/;

Vue.component("modifying-tags", {
    props: {
        name: String,
        color: String,
        isNewTag: String // only new tag
    },
    data: function() {
        return {
            newName: this.name,
            newColor: this.color,
            isChanged: this.isNewTag !== true ? false : true, // isNewTag wasn't passed,
            isError: false,
            isDeleted: false
        };
    },
    destroyed: function() {
        // We delete a tag only after closing the window
        // It lets us to undo the file deleting
        if (this.isDeleted) {
            // this.$parent.tagsAPI().delete(this.name);
        }
    },
    methods: {
        check: function() {
            if (
                this.name == this.newName &&
                this.color == this.newColor &&
                isNewTag !== true
            ) {
                // Can skip, if name and color weren't changed
                this.isChanged = false;
                this.isError = false;
                return;
            }
            this.isChanged = true;

            if (
                this.newName.length == 0 ||
                validTagName.exec(this.newName) === null
            ) {
                this.isError = true;
                return;
            }
            if (validColor.exec(this.newColor) === null) {
                this.isError = true;
                return;
            }

            this.isError = false;
        },
        generateRandomColor: function() {
            if (this.isDeleted) {
                return;
            }
            this.isChanged = true;
            this.isError = false; // we can't generate an invalid color
            this.newColor =
                "#" + Math.floor(Math.random() * 16777215).toString(16);
        },
        // API
        save: function() {
            if (this.isError || !this.isChanged) {
                return;
            }

            if (this.isNewTag) {
                // Need to create, not to change
                // this.$parent.tagsAPI().add(this.newName, this.newColor);
            } else {
                // this.$parent.tagsAPI().change(this.newName, this.newColor);
            }

            this.isChanged = false;
        },
        del: function() {
            if (this.isNewTag) {
                // Delete tag right now
                this.$parent.tagsAPI().delNewTag();
                return;
            }

            this.isDeleted = true;
        },
        recover: function() {
            this.isDeleted = false;
        }
    },
    template: `
	<div style="display: inline-flex; margin-bottom: 5px; width: 95%;">
		<div style="width: 2px; height: 20px; margin-right: 3px;" class="vertically">
			<div v-if="isDeleted" style="height: 20px; border-left: 2px solid white;"></div>
			<div v-else-if="isError"	style="height: 20px; border-left: 2px solid red;"></div>
			<div v-else-if="isChanged" style="height: 20px; border-left: 2px solid blue;"></div>
		</div>
		
		<div style="width: 35%; display: flex;">
			<div :style="{ 'background-color': newColor }" class="tag">
				<div>{{newName}}</div>
			</div>
		</div>

		<input @input="check" type="text" maxlength="24" :disabled="isDeleted" v-model="newName" style="width: 35%; margin-right: 10px;">

		<input @input="check" type="text" :disabled="isDeleted" v-model="newColor" style="width: 15%; margin-right: 5px;">

		<i class="material-icons btn"
			title="Generate a new color"
			@click="generateRandomColor"
			style="margin-right: 10px;"
			:style="[isDeleted ? {'opacity': '0.3', 'background-color': 'white', 'cursor': 'default'} : {'opacity': '1'}]">cached</i>

		<div style="display: flex;">
			<i class="material-icons btn" title="Save" @click="save" style="margin-right: 5px;" 
			:style="[isError || isDeleted || !this.isChanged ? {'opacity': '0.3', 'background-color': 'white', 'cursor': 'default'} : {'opacity': '1'}]">done</i>

			<i v-if="!isDeleted"
				class="material-icons btn"
				title="Delete"
				@click="del"
				:style="[isDeleted ? {'opacity': '0.3', 'background-color': 'white', 'cursor': 'default'} : {'opacity': '1'}]"
			>delete</i>
			<i v-else
				class="material-icons btn"
				title="Undo"
				@click="recover"
			>undo</i>
		</div>
	</div>`
});

// Files in Main block
Vue.component("files", {
    props: ["file", "allTags"],
    data: function() {
        return {
            hover: false
        };
    },
    methods: {
        showContextMenu: function(event, fileData) {
            this.$parent.showContextMenu(event, fileData);
        }
    },
    template: `
	<tr
		:style="[hover ? {'background-color': 'rgba(0, 0, 0, 0.1)'} : {'background-color': 'white'} ]"
		@mouseover="hover = true;"
		@mouseleave="hover = false;"
		@click.right.prevent="showContextMenu(event, file);"
		:title="file.description"
	>
		<td v-if="file.type == 'image'" style="width: 30px;">
			<img :src="file.preview" style="width: 30px;">
		</td>
		<td v-else style="width: 30px; text-align: center;">
			<img :src="'/ext/' + file.filename.split('.').pop()" style="width: 30px;">
		</td>	
		<td style="width: 200px;">
			<div class="filename" :title="file.filename">
				{{file.filename}}
			</div>
		</td>
		<td>
			<div style="display: flex;">
				<file-tag
					v-for="id in file.tags"
					:tag="allTags[id]">
				</file-tag>
			</div>
		</td>
		<td>{{(file.size / (1024 * 1024)).toFixed(1)}}</td>
		<td>{{file.addTime}}</td>
	</tr>`
});
