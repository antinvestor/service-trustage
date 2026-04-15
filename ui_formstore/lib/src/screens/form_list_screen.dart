import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:antinvestor_ui_core/widgets/entity_list_page.dart';

import '../models/form_definition.dart';
import '../providers/formstore_providers.dart';
import '../widgets/form_definition_tile.dart';

/// Screen listing all form definitions with search and create action.
class FormListScreen extends ConsumerStatefulWidget {
  const FormListScreen({super.key});

  @override
  ConsumerState<FormListScreen> createState() => _FormListScreenState();
}

class _FormListScreenState extends ConsumerState<FormListScreen> {
  String _search = '';
  bool? _activeFilter;

  @override
  Widget build(BuildContext context) {
    final asyncDefs = ref.watch(formDefinitionListProvider(_activeFilter));

    return asyncDefs.when(
      loading: () => EntityListPage<FormDefinition>(
        title: 'Forms',
        icon: Icons.description_outlined,
        items: const [],
        isLoading: true,
        itemBuilder: (_, _) => const SizedBox.shrink(),
        searchHint: 'Search forms...',
        onSearchChanged: (v) => setState(() => _search = v),
        actionLabel: 'New Form',
        onAction: () => context.go('/formstore/create'),
        filterWidget: _buildFilterChip(),
      ),
      error: (err, _) => EntityListPage<FormDefinition>(
        title: 'Forms',
        icon: Icons.description_outlined,
        items: const [],
        error: err.toString(),
        onRetry: () => ref.invalidate(formDefinitionListProvider),
        itemBuilder: (_, _) => const SizedBox.shrink(),
        filterWidget: _buildFilterChip(),
      ),
      data: (definitions) {
        final filtered = _search.isEmpty
            ? definitions
            : definitions
                .where((d) =>
                    d.name.toLowerCase().contains(_search.toLowerCase()) ||
                    d.formId.toLowerCase().contains(_search.toLowerCase()))
                .toList();

        return EntityListPage<FormDefinition>(
          title: 'Forms',
          icon: Icons.description_outlined,
          items: filtered,
          itemBuilder: (context, def) => FormDefinitionTile(
            definition: def,
            onTap: () => context.go('/formstore/detail/${def.id}', extra: def),
          ),
          searchHint: 'Search forms...',
          onSearchChanged: (v) => setState(() => _search = v),
          actionLabel: 'New Form',
          onAction: () => context.go('/formstore/create'),
          filterWidget: _buildFilterChip(),
        );
      },
    );
  }

  Widget _buildFilterChip() {
    return SegmentedButton<bool?>(
      segments: const [
        ButtonSegment(value: null, label: Text('All')),
        ButtonSegment(value: true, label: Text('Active')),
        ButtonSegment(value: false, label: Text('Inactive')),
      ],
      selected: {_activeFilter},
      onSelectionChanged: (v) => setState(() => _activeFilter = v.first),
      showSelectedIcon: false,
      style: const ButtonStyle(
        visualDensity: VisualDensity.compact,
      ),
    );
  }
}
