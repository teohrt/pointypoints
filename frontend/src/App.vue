<template>
  <div id="app">
    <main-nav />
    <div>
      <router-view/>
    </div>
    <b-modal v-model="hasRemoteError" role="alert" ok-only id="remoteErrorDialog" title="Something went wrong">
      <p id="remoteErrorMessage">Please try again.</p>
    </b-modal>
  </div>
</template>

<script lang="ts">
import { Vue, Component, Watch } from 'vue-property-decorator'
import MainNav from './navigation/MainNav.vue'
import { AppStore } from '@/app/AppStore'
import { PointingSessionStore } from '@/pointing/PointingSessionStore'

@Component({
  components: {
    MainNav
  }
})
export default class App extends Vue {
  created() {
    this.$store.dispatch(PointingSessionStore.ACTION_INITIALIZE)
  }

  get sessionActive(): boolean {
    return this.$store.state.pointingSession.sessionActive
  }

  get hasRemoteError(): boolean {
    return this.$store.state.app.errorToAck != null
  }

  set hasRemoteError(error) {
    if (!error) {
      this.$store.dispatch(AppStore.ACTION_ACK_REMOTE_ERROR)
    }
  }
}
</script>

<style lang="scss">
@import "~bootstrap/scss/bootstrap";
html {
  position: relative;
  min-height: 100%;
}
</style>
