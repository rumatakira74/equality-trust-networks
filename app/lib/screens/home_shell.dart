// Главная навигация (этап 7): Кошелёк / Поручительства / Реестр /
// Настройки.
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../state/app_state.dart';
import 'migration/migration_screen.dart';
import 'registry/registry_screen.dart';
import 'settings/settings_screen.dart';
import 'vouching/vouching_screen.dart';
import 'wallet/wallet_screen.dart';

class HomeShell extends StatefulWidget {
  const HomeShell({super.key});

  @override
  State<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends State<HomeShell> {
  int _tab = 0;

  @override
  void initState() {
    super.initState();
    // узел живёт всё время приложения (foreground service, 12.2)
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final app = context.read<AppState>();
      if (!app.nodeRunning) app.startNode();
      app.refreshWallet();
    });
  }

  @override
  Widget build(BuildContext context) {
    final screens = [
      const WalletScreen(),
      const VouchingScreen(),
      const RegistryScreen(),
      const MigrationScreen(),
      const SettingsScreen(),
    ];
    return Scaffold(
      body: IndexedStack(index: _tab, children: screens),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tab,
        onDestinationSelected: (i) => setState(() => _tab = i),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.account_balance_wallet), label: 'Кошелёк'),
          NavigationDestination(icon: Icon(Icons.handshake), label: 'Поручительства'),
          NavigationDestination(icon: Icon(Icons.list_alt), label: 'Реестр'),
          NavigationDestination(icon: Icon(Icons.switch_account), label: 'Миграция'),
          NavigationDestination(icon: Icon(Icons.settings), label: 'Настройки'),
        ],
      ),
    );
  }
}
