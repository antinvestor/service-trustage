import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// Dark code block for displaying JSON payloads with copy support.
class JsonBlock extends StatelessWidget {
  const JsonBlock({
    super.key,
    this.label,
    required this.value,
    this.compact = false,
  });

  final String? label;
  final Map<String, dynamic>? value;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final json = value != null
        ? const JsonEncoder.withIndent('  ').convert(value)
        : '{}';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (label != null) ...[
          Row(
            children: [
              Text(
                label!,
                style: theme.textTheme.labelSmall?.copyWith(
                  fontWeight: FontWeight.w600,
                  letterSpacing: 0.3,
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.copy, size: 14),
                onPressed: () => Clipboard.setData(ClipboardData(text: json)),
                tooltip: 'Copy',
                visualDensity: VisualDensity.compact,
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(minWidth: 28, minHeight: 28),
              ),
            ],
          ),
          const SizedBox(height: 4),
        ],
        Container(
          width: double.infinity,
          constraints: BoxConstraints(maxHeight: compact ? 120 : 300),
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: const Color(0xFF1C1917), // stone-950
            borderRadius: BorderRadius.circular(10),
          ),
          child: SingleChildScrollView(
            child: SelectableText(
              json,
              style: const TextStyle(
                fontFamily: 'monospace',
                fontSize: 12,
                color: Color(0xFFE7E5E4), // stone-200
                height: 1.5,
              ),
            ),
          ),
        ),
      ],
    );
  }
}
