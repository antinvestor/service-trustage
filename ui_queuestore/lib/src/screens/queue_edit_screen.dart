import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../models/queue_definition.dart';
import '../providers/queuestore_providers.dart';

/// Create or edit a queue definition.
class QueueEditScreen extends ConsumerStatefulWidget {
  const QueueEditScreen({super.key, this.queue});

  final QueueDefinition? queue;

  @override
  ConsumerState<QueueEditScreen> createState() => _QueueEditScreenState();
}

class _QueueEditScreenState extends ConsumerState<QueueEditScreen> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _nameCtl;
  late final TextEditingController _descCtl;
  late final TextEditingController _priorityCtl;
  late final TextEditingController _capacityCtl;
  late final TextEditingController _slaCtl;
  bool _isSaving = false;

  bool get _isEditing => widget.queue != null;

  @override
  void initState() {
    super.initState();
    final q = widget.queue;
    _nameCtl = TextEditingController(text: q?.name ?? '');
    _descCtl = TextEditingController(text: q?.description ?? '');
    _priorityCtl =
        TextEditingController(text: (q?.priorityLevels ?? 3).toString());
    _capacityCtl =
        TextEditingController(text: (q?.maxCapacity ?? 0).toString());
    _slaCtl = TextEditingController(text: (q?.slaMinutes ?? 30).toString());
  }

  @override
  void dispose() {
    _nameCtl.dispose();
    _descCtl.dispose();
    _priorityCtl.dispose();
    _capacityCtl.dispose();
    _slaCtl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(_isEditing ? 'Edit Queue' : 'New Queue'),
        actions: [
          FilledButton.icon(
            onPressed: _isSaving ? null : _save,
            icon: _isSaving
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.save, size: 18),
            label: Text(_isSaving ? 'Saving...' : 'Save'),
          ),
          const SizedBox(width: 12),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Form(
          key: _formKey,
          child: Card(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
              side: BorderSide(color: theme.colorScheme.outlineVariant),
            ),
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  TextFormField(
                    controller: _nameCtl,
                    decoration: const InputDecoration(
                      labelText: 'Queue Name',
                      hintText: 'e.g., Customer Service',
                    ),
                    validator: (v) =>
                        (v == null || v.isEmpty) ? 'Required' : null,
                  ),
                  const SizedBox(height: 16),
                  TextFormField(
                    controller: _descCtl,
                    decoration: const InputDecoration(
                      labelText: 'Description',
                      hintText: 'Optional description',
                    ),
                    maxLines: 2,
                  ),
                  const SizedBox(height: 16),
                  Row(
                    children: [
                      Expanded(
                        child: TextFormField(
                          controller: _priorityCtl,
                          decoration: const InputDecoration(
                            labelText: 'Priority Levels',
                          ),
                          keyboardType: TextInputType.number,
                          validator: (v) {
                            if (v == null || v.isEmpty) return 'Required';
                            final n = int.tryParse(v);
                            if (n == null || n < 1) return 'Min 1';
                            return null;
                          },
                        ),
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: TextFormField(
                          controller: _capacityCtl,
                          decoration: const InputDecoration(
                            labelText: 'Max Capacity',
                            helperText: '0 = unlimited',
                          ),
                          keyboardType: TextInputType.number,
                        ),
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: TextFormField(
                          controller: _slaCtl,
                          decoration: const InputDecoration(
                            labelText: 'SLA (minutes)',
                          ),
                          keyboardType: TextInputType.number,
                          validator: (v) {
                            if (v == null || v.isEmpty) return 'Required';
                            final n = int.tryParse(v);
                            if (n == null || n < 1) return 'Min 1';
                            return null;
                          },
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _isSaving = true);

    try {
      final notifier = ref.read(queueDefinitionNotifierProvider.notifier);

      if (_isEditing) {
        await notifier.update(widget.queue!.id, {
          'name': _nameCtl.text,
          'description': _descCtl.text,
          'priority_levels': int.parse(_priorityCtl.text),
          'max_capacity': int.tryParse(_capacityCtl.text) ?? 0,
          'sla_minutes': int.parse(_slaCtl.text),
        });
      } else {
        final def = QueueDefinition(
          id: '',
          name: _nameCtl.text,
          description: _descCtl.text,
          priorityLevels: int.parse(_priorityCtl.text),
          maxCapacity: int.tryParse(_capacityCtl.text) ?? 0,
          slaMinutes: int.parse(_slaCtl.text),
        );
        await notifier.create(def);
      }

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content:
                  Text(_isEditing ? 'Queue updated' : 'Queue created')),
        );
        context.go('/queuestore');
      }
    } catch (e) {
      if (mounted) {
        setState(() => _isSaving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Save failed: $e')),
        );
      }
    }
  }
}
