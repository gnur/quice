<template>
  <div class="tile is-ancestor is-10">

    <div v-for="playlist in playlists" :key="playlist" class="tile is-parent">
      <router-link :to="{ name: 'VideoPlayer', params: { user: user, playlist: playlist }}" class="tile is-child box">
          <p class="title">{{ playlist }}</p>
        </router-link>
      </div>

  </div>
</template>

<script>
import axios from "axios";

export default {
  name: "PlaylistSelect",
  props: ["user"],
  data: function() {
    return {
      playlists: []
    };
  },
  methods: {
    getUsers: function() {
      axios.get("/api/playlists/" + this.user + "/").then(
        response => {
          this.playlists = response.data.playlists;
        },
        error => {
          console.log(error);
        }
      );
    }
  },
  mounted: function() {
    this.getUsers();
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
h1,
h2 {
  font-weight: normal;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}
a {
  color: #42b983;
}
</style>
