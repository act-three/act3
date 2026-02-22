import "./turbo.es2017-esm.js";
import { Application } from "./stimulus.js";
import DialogController from "../view/dialog-controller.js";
import PlayerController from "../view/player-controller.js";
import ListController from "./list/list-controller.js";
import SidebarController from "./sidebar/sidebar-controller.js";
import AddTorrentController from "../view/add-torrent-controller.js";

window.Stimulus = Application.start();
Stimulus.register("dialog", DialogController);
Stimulus.register("player", PlayerController);
Stimulus.register("list", ListController);
Stimulus.register("sidebar", SidebarController);
Stimulus.register("add-torrent", AddTorrentController);
