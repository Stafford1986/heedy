var settingsPageMap = {};
var settingsRoutes = [];

class Settings {
    constructor(app) {
        this.store = app.store;
    }
    addPage(p) {
        settingsPageMap[p.path] = p;
    }
    $onInit() {
        Object.values(settingsPageMap).forEach((r) => {
            if (r.path.startsWith("/")) {
                r.path = r.path.substring(1, r.path.length);
            }
            settingsRoutes.push(r);
        });
        this.store.commit("setSettingsRoutes", settingsRoutes);
    }
}

export {
    settingsRoutes
};
export default Settings;