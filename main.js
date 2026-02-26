import "./web/turbo.es2017-esm.js";
import { Application } from "./web/stimulus.js";
import DialogController from "./view/dialog-controller.js";
import PlayerController from "./view/player-controller.js";
import ListController from "./ui/list.js";
import SidebarController from "./web/sidebar/sidebar-controller.js";
import AddTorrentController from "./view/add-torrent-controller.js";

window.Stimulus = Application.start();
Stimulus.register("dialog", DialogController);
Stimulus.register("player", PlayerController);
Stimulus.register("list", ListController);
Stimulus.register("sidebar", SidebarController);
Stimulus.register("add-torrent", AddTorrentController);
