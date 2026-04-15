import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../models/queue_item.dart';
import '../providers/queuestore_providers.dart';

/// Screen for adding a new item to a queue.
class EnqueueScreen extends ConsumerStatefulWidget {
  const EnqueueScreen({super.key, required this.queueId});

  final String queueId;

  @override
  ConsumerState<EnqueueScreen> createState() => _EnqueueScreenState();
}

class _EnqueueScreenState extends ConsumerState<EnqueueScreen> {
  final _formKey = GlobalKey<FormState>();
  final _customerIdCtl = TextEditingController();
  final _categoryCtl = TextEditingController();
  final _ticketNoCtl = TextEditingController();
  int _priority = 1;
  bool _isSaving = false;

  @override
  void dispose() {
    _customerIdCtl.dispose();
    _categoryCtl.dispose();
    _ticketNoCtl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Add to Queue'),
        actions: [
          FilledButton.icon(
            onPressed: _isSaving ? null : _enqueue,
            icon: _isSaving
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.person_add, size: 18),
            label: Text(_isSaving ? 'Adding...' : 'Add'),
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
                    controller: _customerIdCtl,
                    decoration: const InputDecoration(
                      labelText: 'Customer ID',
                      hintText: 'Optional identifier',
                    ),
                  ),
                  const SizedBox(height: 16),
                  TextFormField(
                    controller: _ticketNoCtl,
                    decoration: const InputDecoration(
                      labelText: 'Ticket Number',
                      hintText: 'Optional ticket number',
                    ),
                  ),
                  const SizedBox(height: 16),
                  TextFormField(
                    controller: _categoryCtl,
                    decoration: const InputDecoration(
                      labelText: 'Category',
                      hintText: 'e.g., General, VIP',
                    ),
                  ),
                  const SizedBox(height: 16),
                  Text('Priority', style: theme.textTheme.titleSmall),
                  const SizedBox(height: 8),
                  SegmentedButton<int>(
                    segments: const [
                      ButtonSegment(value: 1, label: Text('Normal')),
                      ButtonSegment(value: 2, label: Text('High')),
                      ButtonSegment(value: 3, label: Text('Urgent')),
                    ],
                    selected: {_priority},
                    onSelectionChanged: (v) =>
                        setState(() => _priority = v.first),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _enqueue() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _isSaving = true);

    try {
      final item = QueueItem(
        id: '',
        queueId: widget.queueId,
        priority: _priority,
        customerId: _customerIdCtl.text,
        ticketNo: _ticketNoCtl.text,
        category: _categoryCtl.text,
      );

      await ref
          .read(queueItemNotifierProvider.notifier)
          .enqueue(widget.queueId, item);

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Added to queue')),
        );
        context.go('/queuestore/detail/${widget.queueId}');
      }
    } catch (e) {
      if (mounted) {
        setState(() => _isSaving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Enqueue failed: $e')),
        );
      }
    }
  }
}
