<template>
  <section>
    <b-table
      :data="videos"
      :checked-rows.sync="checkedRows"
      paginated
      striped
      checkable
      per-page="50"
    >
      <template slot-scope="props">
        <b-table-column field="name" label="name">{{ props.row.filename }}</b-table-column>
        <b-table-column field="changed" label="changed">{{ formatDateTime(props.row.changed) }}</b-table-column>
        <b-table-column field="completed" label="watched">
          <a @click="toggleVideoStatus(props.row.key)">{{ props.row.completed }}</a>
        </b-table-column>
      </template>
    </b-table>
  </section>
</template>

<script>
import axios from "axios";

export default {
  name: "VideoPlayer",
  props: ["user", "playlist"],
  data: function() {
    return {
      videos: [],
      current: ""
    };
  },
  filters: {
    keyToNice(value) {
      if (!value) return "";
      return value.replace(/(_|\/)/g, " ");
    }
  },
  methods: {
    formatDateTime(dateStr) {
      return dateStr.replace("T", " ").substr(0, 16);
    },
    toggleVideoStatus: function(key) {
      var vm = this;
      vm.loaded = false;
      axios
        .post("/api/togglecompleted/", {
          user: vm.user,
          key: key,
          playlist: vm.playlist
        })
        .then(
          response => {
            this.loadVideos();
          },
          error => {
            console.log(error);
          }
        );
    },
    loadVideos: function() {
      axios.get("/api/current/" + this.user + "/" + this.playlist + "/").then(
        response => {
          var resp = response.data;
          this.videos = [];
          for (let v of resp.all.reverse()) {
            var vid = resp.videos[v];
            vid.key = v;
            this.videos.push(resp.videos[v]);
          }
          var vm = this;
        },
        error => {
          console.log(error);
        }
      );
    }
  },
  mounted: function() {
    this.loadVideos();
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
</style>
