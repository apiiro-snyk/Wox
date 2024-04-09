import 'package:fluent_ui/fluent_ui.dart';
import 'package:wox/entity/wox_plugin_setting_newline.dart';

import 'wox_setting_plugin_item_view.dart';

class WoxSettingPluginNewLine extends WoxSettingPluginItem {
  final PluginSettingValueNewLine item;

  const WoxSettingPluginNewLine(super.plugin, this.item, super.onUpdate, {super.key, required});

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.only(top: 4, bottom: 4),
      child: Row(
        children: [
          Expanded(
              child: SizedBox(
            width: 1,
          )),
        ],
      ),
    );
  }
}
