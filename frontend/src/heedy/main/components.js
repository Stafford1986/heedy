/**
 * This file contains the main components that are used in Heedy's core.
 * The components here will be stable over minor versions, so plugins
 * are free to use them for their own UI
 */

import Loading from "./components/loading.vue";
import Icon from "./components/icon.vue";
import IconEditor from "./components/icon_editor.vue";
import ScopeEditor from "./components/scope_editor.vue";
import PageContainer from "./components/page_container.vue";
import CardPage from "./components/card_page.vue";
import NotFound from "./components/404.vue";
import Header from "./components/header.vue";

function register(Vue) {
    Vue.component("h-loading", Loading);
    Vue.component("h-icon", Icon);
    Vue.component("h-icon-editor", IconEditor);
    Vue.component("h-scope-editor", ScopeEditor);
    Vue.component("h-page-container", PageContainer);
    Vue.component("h-card-page", CardPage);
    Vue.component("h-not-found", NotFound);
    Vue.component("h-header", Header)
}

export {
    Loading,
    Icon,
    IconEditor,
    ScopeEditor,
    PageContainer,
    CardPage,
    NotFound,
    Header
}

export default register;