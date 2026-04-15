import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../models/form_definition.dart';
import '../providers/formstore_providers.dart';

/// Create or edit a form definition.
class FormEditScreen extends ConsumerStatefulWidget {
  const FormEditScreen({
    super.key,
    this.definition,
  });

  final FormDefinition? definition;

  @override
  ConsumerState<FormEditScreen> createState() => _FormEditScreenState();
}

class _FormEditScreenState extends ConsumerState<FormEditScreen> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _formIdCtl;
  late final TextEditingController _nameCtl;
  late final TextEditingController _descCtl;
  late final TextEditingController _schemaCtl;
  late bool _active;
  bool _isSaving = false;

  bool get _isEditing => widget.definition != null;

  @override
  void initState() {
    super.initState();
    final def = widget.definition;
    _formIdCtl = TextEditingController(text: def?.formId ?? '');
    _nameCtl = TextEditingController(text: def?.name ?? '');
    _descCtl = TextEditingController(text: def?.description ?? '');
    _schemaCtl = TextEditingController(
      text: def?.jsonSchema != null
          ? const JsonEncoder.withIndent('  ').convert(def!.jsonSchema)
          : '',
    );
    _active = def?.active ?? true;
  }

  @override
  void dispose() {
    _formIdCtl.dispose();
    _nameCtl.dispose();
    _descCtl.dispose();
    _schemaCtl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(_isEditing ? 'Edit Form' : 'New Form'),
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
                    controller: _formIdCtl,
                    decoration: const InputDecoration(
                      labelText: 'Form ID',
                      hintText: 'unique-form-identifier',
                    ),
                    enabled: !_isEditing,
                    validator: (v) =>
                        (v == null || v.isEmpty) ? 'Required' : null,
                  ),
                  const SizedBox(height: 16),
                  TextFormField(
                    controller: _nameCtl,
                    decoration: const InputDecoration(
                      labelText: 'Name',
                      hintText: 'Form display name',
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
                  SwitchListTile(
                    title: const Text('Active'),
                    value: _active,
                    onChanged: (v) => setState(() => _active = v),
                    contentPadding: EdgeInsets.zero,
                  ),
                  const SizedBox(height: 16),
                  TextFormField(
                    controller: _schemaCtl,
                    decoration: const InputDecoration(
                      labelText: 'JSON Schema',
                      hintText: '{"type": "object", "properties": {...}}',
                      alignLabelWithHint: true,
                    ),
                    maxLines: 10,
                    style: theme.textTheme.bodySmall?.copyWith(
                      fontFamily: 'monospace',
                    ),
                    validator: (v) {
                      if (v == null || v.isEmpty) return null;
                      try {
                        jsonDecode(v);
                        return null;
                      } catch (_) {
                        return 'Invalid JSON';
                      }
                    },
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
      final notifier = ref.read(formDefinitionNotifierProvider.notifier);
      Map<String, dynamic>? schema;
      if (_schemaCtl.text.isNotEmpty) {
        schema = jsonDecode(_schemaCtl.text) as Map<String, dynamic>;
      }

      if (_isEditing) {
        await notifier.update(widget.definition!.id, {
          'name': _nameCtl.text,
          'description': _descCtl.text,
          'active': _active,
          if (schema != null) 'json_schema': schema,
        });
      } else {
        final def = FormDefinition(
          id: '',
          formId: _formIdCtl.text,
          name: _nameCtl.text,
          description: _descCtl.text,
          active: _active,
          jsonSchema: schema,
        );
        await notifier.create(def);
      }

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(_isEditing ? 'Form updated' : 'Form created')),
        );
        context.go('/formstore');
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
